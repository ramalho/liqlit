package markov

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name    string
		order   int
		wantErr bool
	}{
		{"ordem negativa", -1, true},
		{"ordem zero", 0, true},
		{"ordem 1", 1, false},
		{"ordem 2", 2, false},
		{"ordem 5", 5, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain, err := New(tc.order)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("New(%d) devolveu erro nil; queria erro", tc.order)
				}
				return
			}
			if err != nil {
				t.Fatalf("New(%d) devolveu erro %v; queria nil", tc.order, err)
			}
			if got := chain.Order(); got != tc.order {
				t.Errorf("Order() = %d; queria %d", got, tc.order)
			}
		})
	}
}

func TestFeedTable(t *testing.T) {
	testCases := []struct {
		name    string
		order   int
		text    string
		want    map[string][]rune
		wantErr bool
	}{
		{
			name:  "banana ordem 2",
			order: 2,
			text:  "banana",
			want:  map[string][]rune{"ba": {'n'}, "an": {'a', 'a'}, "na": {'n'}},
		},
		{
			name:  "abcabcabc ordem 2",
			order: 2,
			text:  "abcabcabc",
			want:  map[string][]rune{"ab": {'c', 'c', 'c'}, "bc": {'a', 'a'}, "ca": {'b', 'b'}},
		},
		{
			name:  "aba ordem 1",
			order: 1,
			text:  "aba",
			want:  map[string][]rune{"a": {'b'}, "b": {'a'}},
		},
		{
			name:  "acentos contam como uma runa",
			order: 2,
			text:  "açaí",
			want:  map[string][]rune{"aç": {'a'}, "ça": {'í'}},
		},
		{
			name:    "texto menor que a ordem",
			order:   3,
			text:    "ab",
			wantErr: true,
		},
		{
			name:    "texto do tamanho da ordem",
			order:   3,
			text:    "abc",
			wantErr: true,
		},
		{
			name:    "texto vazio",
			order:   1,
			text:    "",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain, err := New(tc.order)
			if err != nil {
				t.Fatalf("New(%d): %v", tc.order, err)
			}
			err = chain.Feed(tc.text)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Feed(%q) devolveu erro nil; queria erro", tc.text)
				}
				return
			}
			if err != nil {
				t.Fatalf("Feed(%q): %v", tc.text, err)
			}
			if !reflect.DeepEqual(chain.table, tc.want) {
				t.Errorf("tabela = %v;\nqueria           %v", chain.table, tc.want)
			}
		})
	}
}

func TestFeedAcumula(t *testing.T) {
	chain, err := New(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, texto := range []string{"ab", "ac"} {
		if err := chain.Feed(texto); err != nil {
			t.Fatalf("Feed(%q): %v", texto, err)
		}
	}
	want := map[string][]rune{"a": {'b', 'c'}}
	if !reflect.DeepEqual(chain.table, want) {
		t.Errorf("tabela = %v; queria %v", chain.table, want)
	}
}

func TestFeedRegistraInicios(t *testing.T) {
	testCases := []struct {
		name  string
		order int
		texts []string
		want  []string
	}{
		{"um texto", 2, []string{"banana"}, []string{"ba"}},
		{"dois textos", 2, []string{"banana", "abacate"}, []string{"ba", "ab"}},
		{"inicio repetido conta duas vezes", 2, []string{"banana", "batata"}, []string{"ba", "ba"}},
		{"acentos", 2, []string{"açaí"}, []string{"aç"}},
		{"texto curto nao registra inicio", 3, []string{"ab"}, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain, err := New(tc.order)
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range tc.texts {
				chain.Feed(text) //nolint:errcheck // erro de texto curto é esperado em um dos casos
			}
			if !reflect.DeepEqual(chain.starts, tc.want) {
				t.Errorf("starts = %q; queria %q", chain.starts, tc.want)
			}
		})
	}
}

// TestGenerateDeterministico usa corpora em que todo prefixo tem um único
// sucessor possível. A saída é então totalmente determinada pela cadeia, e o
// teste não depende do gerador de números aleatórios.
func TestGenerateDeterministico(t *testing.T) {
	testCases := []struct {
		name  string
		order int
		text  string
		n     int
		want  string
	}{
		{"ciclo abc completo", 2, "abcabcabc", 9, "abcabcabc"},
		{"ciclo abc truncado", 2, "abcabcabc", 4, "abca"},
		{"ciclo abc mais longo que o corpus", 2, "abcabcabc", 12, "abcabcabcabc"},
		{"banana", 2, "banana", 6, "banana"},
		{"acentos", 2, "açaí", 4, "açaí"},
		{"n igual a ordem", 2, "abcabcabc", 2, "ab"},
		{"n menor que a ordem", 2, "abcabcabc", 1, "a"},
		{"n zero", 2, "abcabcabc", 0, ""},
		{"n negativo", 2, "abcabcabc", -1, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain, err := New(tc.order)
			if err != nil {
				t.Fatal(err)
			}
			if err := chain.Feed(tc.text); err != nil {
				t.Fatal(err)
			}
			if got := chain.Generate(tc.n); got != tc.want {
				t.Errorf("Generate(%d) = %q; queria %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestGenerateCadeiaVazia(t *testing.T) {
	chain, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	if got := chain.Generate(10); got != "" {
		t.Errorf("Generate(10) em cadeia vazia = %q; queria %q", got, "")
	}
}

// TestGenerateBecoSemSaida cobre corpora em que a cadeia chega a um prefixo
// sem sucessor antes de produzir as n runas pedidas.
func TestGenerateBecoSemSaida(t *testing.T) {
	testCases := []struct {
		name  string
		order int
		text  string
		n     int
		want  string
	}{
		{"corpus minimo", 2, "abc", 10, "abc"},
		{"ordem 1 sem ciclo", 1, "abcd", 100, "abcd"},
		{"para no fim do corpus", 3, "abcdef", 50, "abcdef"},
		{"acentos no fim", 2, "açaí", 50, "açaí"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain, err := New(tc.order)
			if err != nil {
				t.Fatal(err)
			}
			if err := chain.Feed(tc.text); err != nil {
				t.Fatal(err)
			}
			if got := chain.Generate(tc.n); got != tc.want {
				t.Errorf("Generate(%d) = %q; queria %q", tc.n, got, tc.want)
			}
		})
	}
}

// corpusRamificado tem prefixos com mais de um sucessor possível, então a
// saída de Generate depende do sorteio.
const corpusRamificado = "o rato roeu a roupa do rei de roma, e o rei de roma rasgou a roupa do rato"

func alimenta(t *testing.T, order int, text string) *Chain {
	t.Helper()
	chain, err := New(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.Feed(text); err != nil {
		t.Fatal(err)
	}
	return chain
}

func TestSetSeedReproduz(t *testing.T) {
	testCases := []struct {
		name  string
		order int
		seed  uint64
		n     int
	}{
		{"ordem 2 semente 0", 2, 0, 60},
		{"ordem 2 semente 42", 2, 42, 60},
		{"ordem 3 semente 42", 3, 42, 60},
		{"ordem 4 semente 7", 4, 7, 200},
		{"ordem 1 semente 999", 1, 999, 30},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			primeira := alimenta(t, tc.order, corpusRamificado)
			primeira.SetSeed(tc.seed)
			want := primeira.Generate(tc.n)

			segunda := alimenta(t, tc.order, corpusRamificado)
			segunda.SetSeed(tc.seed)
			if got := segunda.Generate(tc.n); got != want {
				t.Errorf("com a semente %d as saídas divergiram:\n%q\n%q", tc.seed, got, want)
			}
		})
	}
}

func TestSetSeedSementesDiferentes(t *testing.T) {
	saidas := map[string]bool{}
	for seed := uint64(1); seed <= 8; seed++ {
		chain := alimenta(t, 2, corpusRamificado)
		chain.SetSeed(seed)
		saidas[chain.Generate(60)] = true
	}
	if len(saidas) < 2 {
		t.Errorf("8 sementes produziram %d saída(s) distinta(s); queria pelo menos 2", len(saidas))
	}
}

// TestGenerateSoUsaRunasDoCorpus garante que o liquidificador não inventa
// caracteres: tudo que sai já estava em alguma amostra.
func TestGenerateSoUsaRunasDoCorpus(t *testing.T) {
	noCorpus := map[rune]bool{}
	for _, r := range corpusRamificado {
		noCorpus[r] = true
	}
	for seed := uint64(1); seed <= 8; seed++ {
		chain := alimenta(t, 3, corpusRamificado)
		chain.SetSeed(seed)
		for _, r := range chain.Generate(300) {
			if !noCorpus[r] {
				t.Fatalf("semente %d gerou a runa %q, que não está no corpus", seed, r)
			}
		}
	}
}
