# samples

Textos de amostra para alimentar o `liqlit`, e as ferramentas que os produziram.

## Amostras prontas

| Arquivo | O que é |
| --- | --- |
| [`o-alienista.pdf`](o-alienista.pdf) | *O Alienista*, de Machado de Assis, em PDF. Domínio público, da Biblioteca Virtual do Estudante Brasileiro (USP). É o PDF citado no [README](../README.md) principal. |
| [`saida-extract.md`](saida-extract.md) | O texto do PDF acima, extraído por [`extract.go`](extract.go). |
| [`o-alienista.md`](o-alienista.md) | O mesmo PDF, extraído por [`convert_pdf.py`](convert_pdf.py). Traz marcação Markdown (títulos, negritos) e junta as linhas quebradas pela diagramação. |
| [`flupy-cap24.adoc`](flupy-cap24.adoc) | Capítulo 24 do *Python Fluente*, em AsciiDoc: prosa técnica com trechos de código, para contrastar com a prosa literária do Machado. |

Qualquer um dos arquivos de texto serve de entrada direta:

```sh
go run ./cmd/liqlit -ordem 6 samples/saida-extract.md
```

## Os dois extratores

Os dois leem o mesmo PDF e escrevem o mesmo texto de jeitos diferentes. Nenhum
é extrator de PDF de uso geral.

### `extract.go`

Escrito só com a biblioteca padrão de Go, é o extrator que o README principal
usa. Escreve na saída padrão:

```sh
go run ./samples samples/o-alienista.pdf | go run ./cmd/liqlit -ordem 6
```

Cobre o caso simples, que é o do PDF deste diretório: fluxos comprimidos com
FlateDecode e fontes com `/WinAnsiEncoding`. Num PDF que fuja disso o texto sai
truncado ou embaralhado. Os detalhes estão no comentário no topo do arquivo.

Note que é `package main` num diretório à parte, por isso `go run ./samples`.

### `convert_pdf.py`

Alternativa em Python, apoiada no
[`pymupdf4llm`](https://pypi.org/project/pymupdf4llm/) (`pip install
pymupdf4llm`). Grava um `.md` ao lado do PDF, em vez de escrever na saída
padrão:

```sh
./convert_pdf.py o-alienista.pdf   # gera o-alienista.md
```

Sem argumento, assume `o-alienista.pdf`. Depois da extração, passa um limpador
que remove hifenização de fim de linha e recompõe os parágrafos.
