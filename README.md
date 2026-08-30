# liqlit

Liquidificador literário, uma brincadeira com cadeia de Markov.

`liqlit` lê textos de amostra, aprende com que frequência cada caractere segue
os anteriores, e depois escreve um texto novo sorteando caractere por caractere
com essas mesmas frequências. O resultado não é português — mas com um pouco de
contexto vai ficando cada vez mais parecido com português.

## Prompt inicial

Este projeto nasceu deste pedido, reproduzido aqui como foi escrito:

> neste repo criaremos liqlit, o Liqudificador Literário. o projeto será escrito
> em Go, e dividido em duas partes: uma biblioteca oferecendo funções para criar
> cadeias de Markov a partir de textos fornecidos, e gerar textos usando as
> cadeias; a segunda parte é uma ferramenta cli chamada liqlit para gerar textos
> a partir de amostras. usaremos TDD: primeiro escrevemos o teste de uma
> features, depois implementamos a feature. é proibido alterar um teste
> previamente escrito sem justificar e pedir autorização. usaremos testes de
> tabela e testes exemplo.

## Instalação

```sh
go install github.com/ramalho/liqlit/cmd/liqlit@latest
```

## Uso da ferramenta

```
Uso: liqlit [opções] [arquivo...]

Opções:
  -ordem int
    	quantas runas de contexto a cadeia usa (default 5)
  -semente uint
    	semente do sorteio, para saída reproduzível
  -tamanho int
    	quantas runas gerar, no máximo (default 500)
```

Sem arquivos, `liqlit` lê a amostra da entrada padrão:

```sh
$ printf 'abcabcabc' | liqlit -ordem 2 -tamanho 12
abcabcabcabc
```

Com arquivos, todos alimentam a mesma cadeia:

```sh
$ liqlit -ordem 6 -tamanho 200 machado/*.txt
```

Passe `-semente` para obter sempre o mesmo texto a partir das mesmas amostras.

### Amostras em PDF

O diretório [`samples/`](samples/) traz um extrator de texto de PDF, escrito só
com a biblioteca padrão, para servir de fonte de amostras:

```sh
go run ./samples samples/o-alienista.pdf | liqlit -ordem 6
```

Ele não é um extrator de uso geral: cobre fluxos `FlateDecode` e fontes com
`/WinAnsiEncoding`, que é o caso do PDF que acompanha o repositório.

### A ordem muda tudo

A ordem é quantos caracteres de contexto a cadeia leva em conta. Com ordem
baixa, sai um borrão fonético; com ordem alta, saem trechos quase inteiros da
amostra. O ponto interessante fica no meio.

Os trechos abaixo saem de *O Alienista*, de Machado de Assis, extraído do PDF
em [`samples/`](samples/). Com `-semente`, cada um é reproduzível:

```sh
go run ./samples samples/o-alienista.pdf > alienista.txt
go run ./cmd/liqlit -ordem 6 -tamanho 170 -semente 1882 alienista.txt
```

**Ordem 3**

```
O Algunta a liminião ter povo. Evaritoso! Era o servação. Não melhora a de Janela, atra.
E não juiz de é ade granco ao bastiu, brio ilusões, os falmoço alguns se o coment
```

**Ordem 4**

```
O Alienista,
polido pelo prestabelecimente feridade do bonitos; mas que
é um defesa do governo anela grande foi pouca
compeu o soube do Mateus
de averá pressão que o nos
```

**Ordem 6**

```
O Alienista, fez agora outros; e depois cerca do seu cuidade, mil outra pedi-lo nem insinuante,— negros, o aspecto tenebroso propuseram-lhe uma procura de Itaguaí, mais r
```

**Ordem 8**

```
O Alienista. João Pina assumia a difícil tarefa do governo;
pareceu-lhe
adequada a um ano, para que manifestou-se nesse lance; confessasse a nenhuma pressão não tem inimi
```

Os quatro começam por "O Alienista" porque a geração parte de um prefixo que
abre algum dos textos alimentados — e aqui só há um.

## Uso da biblioteca

```go
import "github.com/ramalho/liqlit/markov"

chain, err := markov.New(6) // 6 runas de contexto
if err != nil {
	log.Fatal(err)
}
if err := chain.Feed(amostra); err != nil {
	log.Fatal(err)
}
fmt.Println(chain.Generate(500)) // até 500 runas
```

A unidade da cadeia é a runa, não o byte: `"açaí"` tem 4 runas. `Feed` usa o
texto como veio — maiúsculas, acentos e espaços em branco não são normalizados
— e pode ser chamado várias vezes para somar amostras na mesma cadeia.
`Generate` pode devolver menos runas que as pedidas, se a cadeia chegar a um
trecho sem continuação. Para saída reproduzível, chame `SetSeed` antes.

Documentação completa: `go doc github.com/ramalho/liqlit/markov`.

## Desenvolvimento

O projeto é escrito com TDD: cada feature começa por um teste que falha.
**Testes já escritos não são alterados sem justificativa e autorização.**
Os testes são de tabela (`markov/chain_test.go`, `cmd/liqlit/main_test.go`) e
de exemplo (`markov/example_test.go`, que viram documentação no godoc).

```sh
go test ./...
go test -race ./...
go vet ./...
```

## Licença

BSD 3-Clause. Veja [LICENSE](LICENSE).
