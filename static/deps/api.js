var api = api || {};

(function(api) {

function call(method, req, callback) {
  $.ajax({
    url: "/rpc",
    dataType: "json",
    data: JSON.stringify({
      method: method,
      params: [req],
      id: 1
    }),
    contentType: 'application/json',
    timeout: 5000,
    type: 'POST',
    success: function(data, status, XHR) {
      if (data.result) {
        callback(data.result);
      } else if (data.error) {
        console.log(data.error);
      }
    },
    error: function(jqXHR, textStatus, errorThrown) {
      console.log(errorThrown);
    }
  });
}

api.Call = function(method, req, callback) {
	call('Pilot.' + method, req, callback);
};

})(api);