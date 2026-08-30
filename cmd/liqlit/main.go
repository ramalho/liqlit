// Comando liqlit, o Liquidificador Literário: lê textos de amostra, constrói
// uma cadeia de Markov de caracteres e imprime um texto novo gerado com ela.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ramalho/liqlit/markov"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const uso = `liqlit — Liquidificador Literário

Constrói uma cadeia de Markov de caracteres com os textos de amostra dados e
imprime um texto novo gerado a partir dela. Sem arquivos, lê a amostra da
entrada padrão.

Uso: liqlit [opções] [arquivo...]

Opções:
`

// nomeStdin identifica a entrada padrão nas mensagens de erro.
const nomeStdin = "amostra da entrada padrão"

// run é o corpo do comando, com entrada e saída injetadas para poder ser
// testado. Devolve o código de saída do processo.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("liqlit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, uso)
		fs.PrintDefaults()
	}
	ordem := fs.Int("ordem", 5, "quantas runas de contexto a cadeia usa")
	tamanho := fs.Int("tamanho", 500, "quantas runas gerar, no máximo")
	semente := fs.Uint64("semente", 0, "semente do sorteio, para saída reproduzível")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if *tamanho < 1 {
		fmt.Fprintf(stderr, "liqlit: tamanho precisa ser >= 1, recebi %d\n", *tamanho)
		return 1
	}

	chain, err := markov.New(*ordem)
	if err != nil {
		fmt.Fprintf(stderr, "liqlit: %v\n", err)
		return 1
	}
	if semeada(fs) {
		chain.SetSeed(*semente)
	}

	if fs.NArg() == 0 {
		texto, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "liqlit: %s: %v\n", nomeStdin, err)
			return 1
		}
		if err := chain.Feed(string(texto)); err != nil {
			fmt.Fprintf(stderr, "liqlit: %s: %v\n", nomeStdin, err)
			return 1
		}
	}
	for _, caminho := range fs.Args() {
		if err := alimenta(chain, caminho); err != nil {
			fmt.Fprintf(stderr, "liqlit: %s: %v\n", caminho, err)
			return 1
		}
	}

	fmt.Fprintln(stdout, chain.Generate(*tamanho))
	return 0
}

// semeada diz se a flag -semente foi passada na linha de comando. Sem ela,
// cada execução usa uma semente aleatória; com ela, mesmo valendo 0, a saída
// é reproduzível.
func semeada(fs *flag.FlagSet) bool {
	passada := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "semente" {
			passada = true
		}
	})
	return passada
}

// alimenta lê um arquivo inteiro e o acrescenta à cadeia.
func alimenta(chain *markov.Chain, caminho string) error {
	texto, err := os.ReadFile(caminho)
	if err != nil {
		return err
	}
	return chain.Feed(string(texto))
}
