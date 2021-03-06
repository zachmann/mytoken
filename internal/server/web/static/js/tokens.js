
let clipboard = new ClipboardJS('.btn-copy');
clipboard.on('success', function (e) {
    e.clearSelection();
    let el = $(e.trigger);
    let originalText = el.attr('data-original-title');
    el.attr('data-original-title', 'Copied!').tooltip('show');
    el.attr('data-original-title', originalText);
});

$('#get-at').on('click', function(e){
    e.preventDefault();
    let msg = $('#at-msg');
    let copy = $('#at-copy');
    getMT(
        function (res) {
           const mToken = res['mytoken']
            getAT(
                function(tokenRes) {
                    msg.text(tokenRes['access_token']);
                    msg.removeClass('text-danger');
                    copy.removeClass('d-none');
                },
                function (errRes) {
                    msg.text(getErrorMessage(errRes));
                    msg.addClass('text-danger');
                    copy.removeClass('d-none');
                },
                mToken);
        },
        function (errRes) {
            msg.text(getErrorMessage(errRes));
            msg.addClass('text-danger');
            copy.removeClass('d-none');
        }
    );
    return false;
});

function getAT(okCallback, errCallback, mToken) {
    let data = {
        "grant_type": "mytoken",
        "comment": "from web interface"
    };
    if (mToken!==undefined) {
        data["mytoken"]=mToken
    }

    data = JSON.stringify(data);
    $.ajax({
        type: "POST",
        url: storageGet('access_token_endpoint'),
        data: data,
        success: okCallback,
        error: errCallback,
        dataType: "json",
        contentType : "application/json"
    });
}

function getMT(okCallback, errCallback, capability="AT") {
    let data = {
        "name":"mytoken-web MT for "+capability,
        "grant_type": "mytoken",
        "capabilities": [capability],
        "restrictions": [
            {
                "exp":  Math.floor(Date.now() / 1000) + 60,
                "ip": ["this"],
                "usages_AT": capability==="AT" ? 1 : 0,
                "usages_other": capability==="AT" ? 0 : 1
            }
        ]
    };
    data = JSON.stringify(data);
    $.ajax({
        type: "POST",
        url: storageGet('mytoken_endpoint'),
        data: data,
        success: okCallback,
        error: errCallback,
        dataType: "json",
        contentType : "application/json"
    });
}

function revokeMT(callback, recursive=true) {
    let data = {
        "recursive": recursive
    };
    data = JSON.stringify(data);
    $.ajax({
        type: "POST",
        url: storageGet('revocation_endpoint'),
        data: data,
        success: callback,
        error: callback,
        dataType: "json",
        contentType : "application/json"
    });
}
