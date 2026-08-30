package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

// executa roda a CLI com stdin e argumentos dados, devolvendo o que ela
// escreveu em stdout, em stderr, e seu código de saída.
func executa(args []string, stdin string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), code
}

// TestRunDeterministico usa corpora em que todo prefixo tem um único sucessor,
// então a saída não depende do sorteio.
func TestRunDeterministico(t *testing.T) {
	testCases := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{
			name:  "le de stdin quando nao ha arquivos",
			args:  []string{"-ordem", "2", "-tamanho", "9"},
			stdin: "abcabcabc",
			want:  "abcabcabc\n",
		},
		{
			name: "le de um arquivo",
			args: []string{"-ordem", "2", "-tamanho", "9", "testdata/ciclo.txt"},
			want: "abcabcabc\n",
		},
		{
			name: "gera mais runas que o corpus",
			args: []string{"-ordem", "2", "-tamanho", "12", "testdata/ciclo.txt"},
			want: "abcabcabcabc\n",
		},
		{
			// Com ordem 8 o corpus de 9 runas tem uma transição só, e a
			// cadeia morre logo depois dela.
			name: "para no beco sem saida",
			args: []string{"-ordem", "8", "-tamanho", "99", "testdata/ciclo.txt"},
			want: "abcabcabc\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := executa(tc.args, tc.stdin)
			if code != 0 {
				t.Fatalf("código de saída = %d; queria 0. stderr: %s", code, stderr)
			}
			if stdout != tc.want {
				t.Errorf("stdout = %q; queria %q", stdout, tc.want)
			}
		})
	}
}

func TestRunErros(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		stdin   string
		wantErr string // trecho esperado em stderr
	}{
		{"ordem zero", []string{"-ordem", "0"}, "abcabc", "ordem"},
		{"ordem negativa", []string{"-ordem", "-2"}, "abcabc", "ordem"},
		{"tamanho zero", []string{"-tamanho", "0"}, "abcabc", "tamanho"},
		{"arquivo inexistente", []string{"testdata/naoexiste.txt"}, "", "naoexiste.txt"},
		{"arquivo curto demais", []string{"-ordem", "5", "testdata/curto.txt"}, "", "curto.txt"},
		{"stdin vazio", []string{}, "", "amostra"},
		{"flag desconhecida", []string{"-liquidificador"}, "abcabc", "liquidificador"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := executa(tc.args, tc.stdin)
			if code == 0 {
				t.Fatalf("código de saída = 0; queria != 0. stdout: %q", stdout)
			}
			if !strings.Contains(stderr, tc.wantErr) {
				t.Errorf("stderr = %q; queria conter %q", stderr, tc.wantErr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; queria vazio em caso de erro", stdout)
			}
		})
	}
}

// TestRunVariosArquivos confirma que os arquivos alimentam a mesma cadeia:
// a saída começa por um dos dois textos, sorteado entre eles.
func TestRunVariosArquivos(t *testing.T) {
	possiveis := []string{"abcabc\n", "banana\n"}
	vistas := map[string]bool{}
	for i := 0; i < 20; i++ {
		stdout, stderr, code := executa(
			[]string{"-ordem", "2", "-tamanho", "6", "testdata/ciclo.txt", "testdata/banana.txt"}, "")
		if code != 0 {
			t.Fatalf("código de saída = %d; stderr: %s", code, stderr)
		}
		if !slices.Contains(possiveis, stdout) {
			t.Fatalf("stdout = %q; queria um de %q", stdout, possiveis)
		}
		vistas[stdout] = true
	}
	if len(vistas) < 2 {
		t.Errorf("20 execuções produziram só %v; queria os dois começos", vistas)
	}
}

func TestRunSemente(t *testing.T) {
	args := []string{"-ordem", "3", "-tamanho", "200", "-semente", "42", "testdata/roupa.txt"}

	primeira, _, code := executa(args, "")
	if code != 0 {
		t.Fatalf("código de saída = %d", code)
	}
	segunda, _, _ := executa(args, "")
	if primeira != segunda {
		t.Errorf("mesma semente gerou saídas diferentes:\n%q\n%q", primeira, segunda)
	}

	outraSemente := slices.Clone(args)
	outraSemente[5] = "7"
	terceira, _, _ := executa(outraSemente, "")
	if terceira == primeira {
		t.Errorf("sementes 42 e 7 geraram a mesma saída: %q", primeira)
	}
}

func TestRunPadroes(t *testing.T) {
	stdout, stderr, code := executa(nil, "testdata/roupa.txt não é lido aqui, este texto vai por stdin mesmo")
	if code != 0 {
		t.Fatalf("código de saída = %d; stderr: %s", code, stderr)
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("stdout = %q; queria terminar em newline", stdout)
	}
	if n := len([]rune(strings.TrimSuffix(stdout, "\n"))); n == 0 {
		t.Errorf("saída vazia com as flags padrão")
	}
}

func TestRunAjuda(t *testing.T) {
	stdout, stderr, code := executa([]string{"-h"}, "")
	if code != 0 {
		t.Errorf("código de saída = %d; queria 0 para -h", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; a ajuda deve ir para stderr", stdout)
	}
	for _, flag := range []string{"-ordem", "-tamanho", "-semente"} {
		if !strings.Contains(stderr, flag) {
			t.Errorf("ajuda não menciona %s:\n%s", flag, stderr)
		}
	}
}
