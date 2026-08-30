package markov_test

import (
	"fmt"
	"log"

	"github.com/ramalho/liqlit/markov"
)

// Uma cadeia de ordem 2 alimentada com "banana" aprende que depois de "an"
// vem "a" e depois de "na" vem "n" — e fica girando nesse ciclo.
func Example() {
	chain, err := markov.New(2)
	if err != nil {
		log.Fatal(err)
	}
	if err := chain.Feed("banana"); err != nil {
		log.Fatal(err)
	}
	fmt.Println(chain.Generate(10))
	// Output: banananana
}

// New recusa ordem menor que 1: sem contexto não há cadeia.
func ExampleNew() {
	_, err := markov.New(0)
	fmt.Println(err)
	// Output: ordem precisa ser >= 1, recebi 0
}

// A ordem é o número de runas de contexto usadas para sortear a runa seguinte.
func ExampleChain_Order() {
	chain, _ := markov.New(4)
	fmt.Println(chain.Order())
	// Output: 4
}

// Feed recusa textos curtos demais para a ordem da cadeia: são necessárias
// pelo menos Order()+1 runas para registrar uma transição.
func ExampleChain_Feed() {
	chain, _ := markov.New(3)
	fmt.Println(chain.Feed("abc"))
	fmt.Println(chain.Feed("abcd"))
	// Output:
	// texto tem 3 runas; ordem 3 exige pelo menos 4
	// <nil>
}

// Generate para antes das n runas pedidas quando chega a um prefixo sem
// sucessor, isto é, ao fim de um texto de amostra.
func ExampleChain_Generate() {
	chain, _ := markov.New(2)
	chain.Feed("açaí") //nolint:errcheck
	fmt.Println(chain.Generate(100))
	// Output: açaí
}

// Com a mesma semente, cadeias iguais geram o mesmo texto.
func ExampleChain_SetSeed() {
	const amostra = "o rato roeu a roupa do rei de roma"

	gera := func(seed uint64) string {
		chain, _ := markov.New(3)
		chain.Feed(amostra) //nolint:errcheck
		chain.SetSeed(seed)
		return chain.Generate(40)
	}

	fmt.Println(gera(42) == gera(42))
	// Output: true
}
