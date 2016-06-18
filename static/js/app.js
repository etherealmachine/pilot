function play(video) {
	$.post('/controls?action=play&video='+encodeURIComponent(video))
	.fail(function(jqXHR, textStatus, error) {

	});
	window.location = window.location;
}

function stop() {
	$.post('/controls?action=stop')
	.fail(function(jqXHR, textStatus, error) {

	});
	window.location = window.location;
}

function pause() {
	$.post('/controls?action=pause')
	.fail(function(jqXHR, textStatus, error) {

	});
	window.location = window.location;
}

function resume() {
	$.post('/controls?action=resume')
	.fail(function(jqXHR, textStatus, error) {

	});
	window.location = window.location;
}

function createItem(video) {
	var p = document.createElement('p');
	p.appendChild(document.createTextNode(video));
	p.appendChild(document.createElement('br'));
	var playOnTVLink = document.createElement('a');
	playOnTVLink.href = 'javascript:play("'+video+'")';
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