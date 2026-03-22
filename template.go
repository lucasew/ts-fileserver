package fileserver

import (
	"fmt"
	"io"
)

const HTML_PRELUDE = `
<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>ts-fileserver</title>
		<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/sakura.css/css/sakura.css" type="text/css">
    </head>

<body>
<style>
#status > * {
	margin: 0;
}
</style>

<script>
async function upload() {
	const input = document.getElementById("file")
	const status = document.getElementById("status")
	for (const file of input.files) {
		const paragraph = document.createElement('p')
		status.appendChild(paragraph)
	    const {name} = file
	    const url = window.location.toString() + "/" + name
	    const xhr = new XMLHttpRequest()
	    xhr.open('POST', url, true)
	    xhr.upload.onprogress = function(event) {
	      if (event.lengthComputable) {
	          const percentComplete = (event.loaded / event.total) * 100;
	          paragraph.innerText = name + ": " + percentComplete.toFixed(2) + "%"
	      }
		};
	    xhr.onreadystatechange = function() {
	        if (xhr.readyState === 4) {
	          paragraph.innerText = name + ": " + "DONE"
	        }
	    };
	    console.log(file)
	    xhr.send(file)
	}
}
</script>

<input type="file" id="file" multiple /><button onclick="upload()">Upload</button>
<div id="status"></div>
<ul>
`

func (f *FileServer) WriteHTMLPrelude(w io.Writer) {
	fmt.Fprintf(w, "%s", HTML_PRELUDE)
}
