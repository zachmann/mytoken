package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Songmu/prompter"
	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"

	"github.com/zachmann/mytoken/internal/server/config"
	"github.com/zachmann/mytoken/internal/server/db"
	"github.com/zachmann/mytoken/internal/server/db/dbdefinition"
	"github.com/zachmann/mytoken/internal/server/jws"
	"github.com/zachmann/mytoken/internal/server/model"
	event "github.com/zachmann/mytoken/internal/server/supertoken/event/pkg"
	loggerUtils "github.com/zachmann/mytoken/internal/server/utils/logger"
	model2 "github.com/zachmann/mytoken/pkg/model"
)

var genSigningKeyComm commandGenSigningKey
var createDBComm commandCreateDB

func main() {
	config.LoadForSetup()
	loggerUtils.Init()

	parser := flags.NewNamedParser("mytoken", flags.HelpFlag|flags.PassDoubleDash)
	parser.AddCommand("signing-key", "Generates a new signing key", "Generates a new signing key according to the properties specified in the config file and stores it.", &genSigningKeyComm)
	parser.AddCommand("db", "Setups the database", "Setups the database as needed and specified in the config file.", &createDBComm)
	_, err := parser.Parse()
	if err != nil {
		var flagError *flags.Error
		if errors.As(err, &flagError) {
			if flagError.Type == flags.ErrHelp {
				fmt.Println(err)
				os.Exit(0)
			}
		}
		log.WithError(err).Fatal()
		os.Exit(1)
	}

}

type commandGenSigningKey struct{}
type commandCreateDB struct {
	Username string `short:"u" long:"user" default:"root" description:"This username is used to connect to the database to create a new database, database user, and tables."`
	Password string `short:"p" long:"password" description:"The password for the database user"`
}

// Execute implements the flags.Commander interface
func (c *commandGenSigningKey) Execute(args []string) error {
	sk, _, err := jws.GenerateKeyPair()
	if err != nil {
		return err
	}
	str := jws.ExportPrivateKeyAsPemStr(sk)
	filepath := config.Get().Signing.KeyFile
	if err = ioutil.WriteFile(filepath, []byte(str), 0600); err != nil {
		return err
	}
	log.WithField("filepath", filepath).Info("Wrote key to file.")
	return nil
}

// Execute implements the flags.Commander interface
func (c *commandCreateDB) Execute(args []string) error {
	dsn := fmt.Sprintf("%s:%s@%s(%s)/", c.Username, c.Password, "tcp", config.Get().DB.Host)
	if err := db.ConnectDSN(dsn); err != nil {
		return err
	}
	log.WithField("user", c.Username).Debug("Connected to database")
	if err := checkDB(); err != nil {
		return err
	}
	return db.Transact(func(tx *sqlx.Tx) error {
		if err := createDB(tx); err != nil {
			return err
		}
		if err := createUser(tx); err != nil {
			return err
		}
		if err := createTables(tx); err != nil {
			return err
		}
		if err := addPredefinedValues(tx); err != nil {
			return err
		}
		return nil
	})
}

func addPredefinedValues(tx *sqlx.Tx) error {
	for _, attr := range model.Attributes {
		if _, err := tx.Exec(`INSERT IGNORE INTO Attributes (attribute) VALUES(?)`, attr); err != nil {
			return err
		}
	}
	log.WithField("database", config.Get().DB.DB).Debug("Added attribute values")
	for _, evt := range event.AllEvents {
		if _, err := tx.Exec(`INSERT IGNORE INTO Events (event) VALUES(?)`, evt); err != nil {
			return err
		}
	}
	log.WithField("database", config.Get().DB.DB).Debug("Added event values")
	for _, grt := range model2.AllGrantTypes {
		if _, err := tx.Exec(`INSERT IGNORE INTO Grants (grant_type) VALUES(?)`, grt); err != nil {
			return err
		}
	}
	log.WithField("database", config.Get().DB.DB).Debug("Added grant_type values")
	return nil
}

func createTables(tx *sqlx.Tx) error {
	if _, err := tx.Exec(`USE ` + config.Get().DB.DB); err != nil {
		return err
	}
	for _, cmd := range dbdefinition.DDL {
		cmd = strings.TrimSpace(cmd)
		if len(cmd) > 0 && !strings.HasPrefix(cmd, "--") {
			log.Trace(cmd)
			if _, err := tx.Exec(cmd); err != nil {
				return err
			}
		}
	}
	log.WithField("database", config.Get().DB.DB).Debug("Created tables")
	return nil
}

func createDB(tx *sqlx.Tx) error {
	if _, err := tx.Exec(`DROP DATABASE IF EXISTS ` + config.Get().DB.DB); err != nil {
		return err
	}
	log.WithField("database", config.Get().DB.DB).Debug("Dropped database")
	if _, err := tx.Exec(`CREATE DATABASE ` + config.Get().DB.DB); err != nil {
		return err
	}
	log.WithField("database", config.Get().DB.DB).Debug("Created database")
	return nil
}

func createUser(tx *sqlx.Tx) error {
	log.WithField("user", config.Get().DB.User).Debug("Creating user")
	if _, err := tx.Exec(`CREATE USER IF NOT EXISTS '` + config.Get().DB.User + `' IDENTIFIED BY '` + config.Get().DB.Password + `'`); err != nil {
		return err
	}
	log.WithField("user", config.Get().DB.User).Debug("Created user")
	if _, err := tx.Exec(`GRANT INSERT, UPDATE, DELETE, SELECT ON ` + config.Get().DB.DB + `.* TO '` + config.Get().DB.User + `'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`FLUSH PRIVILEGES `); err != nil {
		return err
	}
	log.WithField("user", config.Get().DB.User).WithField("database", config.Get().DB.DB).Debug("Granted privileges")
	return nil
}

func checkDB() error {
	log.WithField("database", config.Get().DB.DB).Debug("Check if database already exists")
	rows, err := db.DB().Query(`SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME=?`, config.Get().DB.DB)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		if !prompter.YesNo("The database already exists. If we continue all data will be deleted. Do you want to continue?", false) {
			os.Exit(1)
		}
	}
	return nil
}