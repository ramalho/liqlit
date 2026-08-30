// Package markov constrói cadeias de Markov de caracteres a partir de textos
// de amostra, e gera textos novos percorrendo essas cadeias.
//
// A unidade da cadeia é a runa: uma cadeia de ordem N usa as N runas
// anteriores como contexto para sortear a runa seguinte.
package markov

import (
	"fmt"
	"math/rand/v2"
)

// Chain é uma cadeia de Markov sobre runas.
// O valor zero não é utilizável: crie cadeias com [New].
type Chain struct {
	order int
	// table mapeia cada prefixo de order runas para as runas observadas
	// logo depois dele, com repetição: quanto mais vezes uma runa segue um
	// prefixo, mais vezes ela aparece na fatia, e maior sua probabilidade.
	table map[string][]rune
	// starts guarda o prefixo inicial de cada texto alimentado, para que a
	// geração possa começar por um começo de verdade.
	starts []string
	// rng sorteia o começo e cada runa seguinte. Trocado por [Chain.SetSeed]
	// quando se quer geração reproduzível.
	rng *rand.Rand
}

// New devolve uma cadeia de ordem order, ou seja, que usa order runas de
// contexto para sortear a runa seguinte. A ordem precisa ser pelo menos 1.
func New(order int) (*Chain, error) {
	if order < 1 {
		return nil, fmt.Errorf("ordem precisa ser >= 1, recebi %d", order)
	}
	return &Chain{
		order: order,
		table: map[string][]rune{},
		rng:   rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}, nil
}

// Order devolve a ordem da cadeia: quantas runas de contexto ela usa.
func (c *Chain) Order() int {
	return c.order
}

// Feed alimenta a cadeia com um texto de amostra, registrando cada transição
// de um prefixo de Order() runas para a runa seguinte. Pode ser chamado
// várias vezes; os textos se somam na mesma cadeia.
//
// O texto é usado como veio: maiúsculas, acentos e espaços em branco não são
// normalizados. Devolve erro se o texto tem menos de Order()+1 runas, pois
// nesse caso não há nenhuma transição a registrar.
func (c *Chain) Feed(text string) error {
	runes := []rune(text)
	if len(runes) < c.order+1 {
		return fmt.Errorf("texto tem %d runas; ordem %d exige pelo menos %d",
			len(runes), c.order, c.order+1)
	}
	c.starts = append(c.starts, string(runes[:c.order]))
	for i := 0; i+c.order < len(runes); i++ {
		prefix := string(runes[i : i+c.order])
		c.table[prefix] = append(c.table[prefix], runes[i+c.order])
	}
	return nil
}

// SetSeed fixa a semente do sorteio, tornando [Chain.Generate] reproduzível:
// duas cadeias alimentadas com os mesmos textos e a mesma semente geram o
// mesmo resultado. Sem chamar SetSeed, cada cadeia começa com uma semente
// aleatória.
func (c *Chain) SetSeed(seed uint64) {
	c.rng = rand.New(rand.NewPCG(seed, seed))
}

// Generate devolve um texto de até n runas, começando por um dos prefixos
// iniciais sorteados entre os textos alimentados e sorteando cada runa
// seguinte entre as observadas depois do prefixo corrente.
//
// O resultado pode ter menos de n runas: a geração para quando cai num
// prefixo sem sucessor, o que acontece no fim de um texto de amostra.
// Numa cadeia vazia, ou com n <= 0, devolve a string vazia.
func (c *Chain) Generate(n int) string {
	if n <= 0 || len(c.starts) == 0 {
		return ""
	}
	out := []rune(c.starts[c.rng.IntN(len(c.starts))])
	if len(out) >= n {
		return string(out[:n])
	}
	for len(out) < n {
		options := c.table[string(out[len(out)-c.order:])]
		if len(options) == 0 {
			break
		}
		out = append(out, options[c.rng.IntN(len(options))])
	}
	return string(out)
}
