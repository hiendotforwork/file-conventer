# File Converter

A small Go web app that converts documents with LibreOffice. Uploads and results are kept in memory where practical; conversion work files live in a request-specific directory under `/dev/shm` (Linux tmpfs) and are removed when the request ends. There is no database, file history, or persistent upload storage.

## Requirements

- Go 1.22 or newer
- LibreOffice (`soffice` on `PATH`)
- Linux with writable `/dev/shm`

On Debian or Ubuntu, install LibreOffice with `sudo apt install libreoffice`. Then run:

```sh
go run ./cmd/server
```

Open `http://localhost:8080`. Set `PORT` locally to change the listening port; `SERVER_PORT` takes precedence for hosts such as Botkeep.

For a full local setup and Botkeep deployment checklist, see [docs/LOCAL_AND_BOTKEEP.md](docs/LOCAL_AND_BOTKEEP.md).

The server rejects uploads if `/dev/shm` is unavailable. Mount `/dev/shm` as tmpfs in containers. It accepts up to 10 files and 25 MiB total per request, converts at most two requests at a time, and stops a conversion after 30 seconds.

## Supported conversions

The output menu shows only formats supported for every selected input:

| Input | Output formats |
| --- | --- |
| DOCX | PDF, TXT, HTML |
| PPTX | PDF |
| XLSX | PDF, HTML, CSV |
| CSV | XLSX, PDF, HTML |
| TXT | DOCX, PDF, HTML |
| HTML | PDF, TXT |
| Markdown | DOCX, PDF, HTML, TXT |

PDF, Markdown, and PPTX are not offered as output formats in this MVP; LibreOffice in the target environment does not expose reliable export filters for those combinations. PDF input is also unsupported. CSV export from a workbook contains its first sheet only.

## Limits

This MVP depends on LibreOffice's import/export filters and does not promise perfect layout preservation. Supported input type checks inspect file signatures for Office files and parse/validate text formats before conversion. Files are never retained after the request, including failed or cancelled conversions.
