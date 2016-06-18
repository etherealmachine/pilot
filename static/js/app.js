function play(video) {
	$.post('/controls?action=play&video='+encodeURIComponent(video))
	.done(function(data, textStatus, jqXHR) {
		window.location = window.location;
	})
	.fail(function(jqXHR, textStatus, error) {
		console.error(jqXHR.responseText);
	});
}

function stop() {
	$.post('/controls?action=stop')
	.done(function(data, textStatus, jqXHR) {
		window.location = window.location;
	})
	.fail(function(jqXHR, textStatus, error) {
		console.error(jqXHR.responseText);
	});
}

function pause() {
	$.post('/controls?action=pause')
	.done(function(data, textStatus, jqXHR) {
		window.location = window.location;
	})
	.fail(function(jqXHR, textStatus, error) {
		console.error(jqXHR.responseText);
	});
}

function resume() {
	$.post('/controls?action=resume')
	.done(function(data, textStatus, jqXHR) {
		window.location = window.location;
	})
	.fail(function(jqXHR, textStatus, error) {
		console.error(jqXHR.responseText);
	});
}

function createItem(video) {
	var p = document.createElement('p');
	p.appendChild(document.createTextNode(video));
	p.appendChild(document.createElement('br'));
	var playOnTVLink = document.createElement('a');
	playOnTVLink.href = 'javascript:play("'+encodeURIComponent(video)+'")';
	playOnTVLink.appendChild(document.createTextNode('Play on TV'));
	p.appendChild(playOnTVLink);
	p.appendChild(document.createElement('br'));
	var playInBrowserLink = document.createElement('a');
	playInBrowserLink.href = '/play?video='+encodeURIComponent(video);
	playInBrowserLink.appendChild(document.createTextNode('Play in Browser'));
	p.appendChild(playInBrowserLink);
	p.appendChild(document.createElement('br'));
	var downloadLink = document.createElement('a');
	downloadLink.href = '/download?video='+encodeURIComponent(video);
	downloadLink.appendChild(document.createTextNode('Download'));
	p.appendChild(downloadLink);
	return p;
}

$(function() {
	$.getJSON('/files.json', function(files) {
		var container = document.getElementById('container');
		files.forEach(function(f) {
			container.appendChild(createItem(f));
		});
	});
});
