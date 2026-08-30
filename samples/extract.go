// O comando extract escreve na saída padrão o texto encontrado num PDF, para
// servir de amostra ao liqlit:
//
//	go run ./samples samples/o-alienista.pdf | go run ./cmd/liqlit -ordem 6
//
// Não é um extrator de PDF de uso geral. Usa só a biblioteca padrão e cobre o
// caso simples, que é o do PDF deste diretório: fluxos comprimidos com
// FlateDecode e fontes com /WinAnsiEncoding, sem CMap /ToUnicode. Num PDF que
// fuja disso — fontes com codificação própria, fluxos com outros filtros — o
// texto sai truncado ou embaralhado.
//
// A ordem do texto segue a ordem dos fluxos no arquivo, que costuma ser a
// ordem das páginas, mas o PDF não obriga a isso.
package main

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("extract: ")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Uso: go run ./samples arquivo.pdf [arquivo.pdf...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	for _, caminho := range flag.Args() {
		texto, err := extractFile(caminho)
		if err != nil {
			log.Fatalf("%s: %v", caminho, err)
		}
		if _, err := io.WriteString(os.Stdout, texto); err != nil {
			log.Fatal(err)
		}
	}
}

// extractFile devolve o texto de todos os fluxos de conteúdo de um PDF.
func extractFile(caminho string) (string, error) {
	raw, err := os.ReadFile(caminho)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return "", errors.New("não começa com %PDF-, não parece um PDF")
	}
	var conteudos [][]byte
	for _, fluxo := range streams(raw) {
		conteudo, err := inflate(fluxo)
		if err != nil {
			continue // fluxo com outro filtro, ou sem compressão: não interessa
		}
		if !isContentStream(conteudo) {
			continue // fontes, imagens, metadados, tabela de referências
		}
		conteudos = append(conteudos, conteudo)
	}
	if len(conteudos) == 0 {
		return "", errors.New("nenhum fluxo de conteúdo legível")
	}
	// Os fluxos são interpretados juntos, e não um a um, porque o PDF permite
	// dividir o conteúdo de uma página em vários fluxos cortando em qualquer
	// ponto — inclusive no meio de um token. Neste PDF um fluxo termina em
	// "...maior )]" e o seguinte começa em "TJ": lidos em separado, essa frase
	// se perderia.
	return extractText(bytes.Join(conteudos, []byte("\n"))), nil
}

// streams devolve, para cada fluxo do arquivo, os bytes a partir do começo dos
// seus dados até o fim do arquivo. Quem acha o fim de cada fluxo é o
// descompressor, que sabe onde o dado comprimido acaba: a palavra endstream
// não serve para isso, porque esses mesmos bytes podem aparecer no meio dos
// dados comprimidos e truncariam o fluxo.
func streams(raw []byte) [][]byte {
	var achados [][]byte
	for i := 0; i < len(raw); {
		j := bytes.Index(raw[i:], []byte("stream"))
		if j < 0 {
			break
		}
		inicio := i + j + len("stream")
		// A palavra-chave stream é seguida de CRLF ou LF, que não fazem parte
		// dos dados.
		if inicio < len(raw) && raw[inicio] == '\r' {
			inicio++
		}
		if inicio < len(raw) && raw[inicio] == '\n' {
			inicio++
		}
		achados = append(achados, raw[inicio:])
		fim := bytes.Index(raw[inicio:], []byte("endstream"))
		if fim < 0 {
			break
		}
		i = inicio + fim + len("endstream")
	}
	return achados
}

// inflate descomprime um fluxo FlateDecode, parando no fim do dado comprimido
// e ignorando o que vier depois. Um fluxo que não descomprima inteiro é
// descartado: meio texto é pior que texto nenhum, porque some sem avisar.
func inflate(data []byte) ([]byte, error) {
	if zr, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
		if out, err := io.ReadAll(zr); err == nil && len(out) > 0 {
			return out, nil
		}
	}
	// Alguns geradores gravam deflate cru, sem o cabeçalho do zlib.
	if out, err := io.ReadAll(flate.NewReader(bytes.NewReader(data))); err == nil && len(out) > 0 {
		return out, nil
	}
	return nil, errors.New("não descomprimiu com Flate")
}

// isContentStream diz se um fluxo descomprimido é conteúdo de página: precisa
// mostrar texto e parecer texto. A segunda condição descarta fontes e imagens,
// que são binárias e podem conter "Tj" por acaso.
func isContentStream(data []byte) bool {
	mostraTexto := bytes.Contains(data, []byte("Tj")) || bytes.Contains(data, []byte("TJ"))
	return mostraTexto && pareceTexto(data)
}

// pareceTexto diz se quase todos os bytes são ASCII imprimível ou espaço,
// como num fluxo de conteúdo, ao contrário de um programa de fonte.
func pareceTexto(data []byte) bool {
	legiveis := 0
	for _, c := range data {
		if (c >= 0x20 && c < 0x7f) || ehEspaco(c) {
			legiveis++
		}
	}
	return len(data) > 0 && float64(legiveis)/float64(len(data)) > 0.9
}

// kernParaEspaco é o recuo mínimo, em milésimos da unidade de texto, que a
// gente lê como separação de palavras num vetor TJ.
const kernParaEspaco = -120

// toleranciaLinha é a variação de altura, em unidades de texto, que ainda
// conta como a mesma linha. Sem ela, um arredondamento de fração de ponto
// parte uma palavra em duas linhas.
const toleranciaLinha = 0.5

// extractText interpreta os operadores de texto de um fluxo de conteúdo.
// Acompanha a posição vertical do cursor para quebrar linhas: cada mudança de
// altura vira uma linha nova na saída.
func extractText(conteudo []byte) string {
	var out, linha strings.Builder
	var pilha []token
	var x, y, entrelinha float64
	sc := &scanner{data: conteudo}

	// quebra fecha a linha corrente, descartando linhas em branco.
	quebra := func() {
		if texto := strings.TrimRight(linha.String(), " "); texto != "" {
			out.WriteString(texto)
			out.WriteByte('\n')
		}
		linha.Reset()
	}
	// vaiPara move o cursor de texto, quebrando a linha se mudou de altura.
	vaiPara := func(nx, ny float64) {
		if math.Abs(ny-y) > toleranciaLinha {
			quebra()
		}
		x, y = nx, ny
	}
	escreve := func(s []byte) {
		linha.WriteString(decodeWinAnsi(s))
	}

	for {
		tk := sc.scan()
		switch tk.kind {
		case tokEOF:
			quebra()
			return out.String()
		case tokOpenArray:
			pilha = pilha[:0]
			continue
		case tokCloseArray, tokSkip:
			continue
		case tokString, tokNumber:
			pilha = append(pilha, tk)
			continue
		}

		n := numeros(pilha)
		switch tk.op {
		case "BT":
			x, y = 0, 0
		case "ET":
			quebra()
		case "Tm":
			if len(n) >= 6 {
				vaiPara(n[4], n[5])
			}
		case "Td":
			if len(n) >= 2 {
				vaiPara(x+n[0], y+n[1])
			}
		case "TD":
			if len(n) >= 2 {
				entrelinha = -n[1]
				vaiPara(x+n[0], y+n[1])
			}
		case "TL":
			if len(n) >= 1 {
				entrelinha = n[0]
			}
		case "T*":
			vaiPara(x, y-entrelinha)
		case "Tj":
			for _, op := range pilha {
				if op.kind == tokString {
					escreve(op.str)
				}
			}
		case "'", "\"":
			vaiPara(x, y-entrelinha)
			for _, op := range pilha {
				if op.kind == tokString {
					escreve(op.str)
				}
			}
		case "TJ":
			for _, op := range pilha {
				switch {
				case op.kind == tokString:
					escreve(op.str)
				case op.num <= kernParaEspaco && !strings.HasSuffix(linha.String(), " "):
					linha.WriteByte(' ')
				}
			}
		}
		pilha = pilha[:0]
	}
}

// numeros extrai os operandos numéricos da pilha, na ordem em que apareceram.
func numeros(pilha []token) []float64 {
	var n []float64
	for _, tk := range pilha {
		if tk.kind == tokNumber {
			n = append(n, tk.num)
		}
	}
	return n
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokString
	tokNumber
	tokOperator
	tokOpenArray
	tokCloseArray
	tokSkip // nomes, dicionários, comentários: irrelevantes para o texto
)

type token struct {
	kind tokenKind
	str  []byte  // tokString
	num  float64 // tokNumber
	op   string  // tokOperator
}

type scanner struct {
	data []byte
	pos  int
}

func ehEspaco(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == 0
}

func ehDelimitador(c byte) bool {
	return bytes.IndexByte([]byte("()<>[]{}/%"), c) >= 0
}

// scan devolve o próximo token do fluxo de conteúdo.
func (s *scanner) scan() token {
	for s.pos < len(s.data) && ehEspaco(s.data[s.pos]) {
		s.pos++
	}
	if s.pos >= len(s.data) {
		return token{kind: tokEOF}
	}
	c := s.data[s.pos]
	switch {
	case c == '(':
		s.pos++
		return token{kind: tokString, str: s.literal()}
	case c == '<':
		if s.pos+1 < len(s.data) && s.data[s.pos+1] == '<' {
			s.pos += 2
			return token{kind: tokSkip}
		}
		s.pos++
		return token{kind: tokString, str: s.hexa()}
	case c == '>':
		s.pos++
		if s.pos < len(s.data) && s.data[s.pos] == '>' {
			s.pos++
		}
		return token{kind: tokSkip}
	case c == '[':
		s.pos++
		return token{kind: tokOpenArray}
	case c == ']':
		s.pos++
		return token{kind: tokCloseArray}
	case c == '/' || c == '{' || c == '}':
		s.pos++
		for s.pos < len(s.data) && !ehEspaco(s.data[s.pos]) && !ehDelimitador(s.data[s.pos]) {
			s.pos++
		}
		return token{kind: tokSkip}
	case c == '%':
		for s.pos < len(s.data) && s.data[s.pos] != '\n' {
			s.pos++
		}
		return token{kind: tokSkip}
	case c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9'):
		inicio := s.pos
		s.pos++
		for s.pos < len(s.data) {
			d := s.data[s.pos]
			if (d >= '0' && d <= '9') || d == '.' || d == '-' || d == '+' {
				s.pos++
				continue
			}
			break
		}
		num, err := strconv.ParseFloat(string(s.data[inicio:s.pos]), 64)
		if err != nil {
			return token{kind: tokSkip}
		}
		return token{kind: tokNumber, num: num}
	default:
		inicio := s.pos
		for s.pos < len(s.data) && !ehEspaco(s.data[s.pos]) && !ehDelimitador(s.data[s.pos]) {
			s.pos++
		}
		return token{kind: tokOperator, op: string(s.data[inicio:s.pos])}
	}
}

// literal lê uma string entre parênteses, já sem os escapes. Os parênteses
// internos podem aninhar, desde que balanceados.
func (s *scanner) literal() []byte {
	var out []byte
	nivel := 1
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		s.pos++
		switch c {
		case '(':
			nivel++
			out = append(out, c)
		case ')':
			nivel--
			if nivel == 0 {
				return out
			}
			out = append(out, c)
		case '\\':
			if s.pos >= len(s.data) {
				return out
			}
			e := s.data[s.pos]
			s.pos++
			switch e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '\n': // barra invertida no fim da linha: continuação
			case '\r':
				if s.pos < len(s.data) && s.data[s.pos] == '\n' {
					s.pos++
				}
			default:
				if e >= '0' && e <= '7' {
					valor := int(e - '0')
					for i := 0; i < 2 && s.pos < len(s.data); i++ {
						d := s.data[s.pos]
						if d < '0' || d > '7' {
							break
						}
						valor = valor*8 + int(d-'0')
						s.pos++
					}
					out = append(out, byte(valor))
					continue
				}
				out = append(out, e) // \( \) \\ e quaisquer outros
			}
		default:
			out = append(out, c)
		}
	}
	return out
}

// hexa lê uma string entre < e >, em pares de dígitos hexadecimais.
func (s *scanner) hexa() []byte {
	var digitos []byte
	for s.pos < len(s.data) && s.data[s.pos] != '>' {
		c := s.data[s.pos]
		s.pos++
		if ehEspaco(c) {
			continue
		}
		digitos = append(digitos, c)
	}
	if s.pos < len(s.data) {
		s.pos++ // fecha o >
	}
	if len(digitos)%2 == 1 {
		digitos = append(digitos, '0') // dígito ímpar final vale como se fosse d0
	}
	out := make([]byte, 0, len(digitos)/2)
	for i := 0; i+1 < len(digitos); i += 2 {
		valor, err := strconv.ParseUint(string(digitos[i:i+2]), 16, 8)
		if err != nil {
			continue
		}
		out = append(out, byte(valor))
	}
	return out
}

// altoWinAnsi são as runas de 0x80 a 0x9F na WinAnsiEncoding, a faixa em que
// ela difere da Latin-1. O zero marca as posições sem caractere atribuído.
var altoWinAnsi = [32]rune{
	'€', 0, '‚', 'ƒ', '„', '…', '†', '‡',
	'ˆ', '‰', 'Š', '‹', 'Œ', 0, 'Ž', 0,
	0, '‘', '’', '“', '”', '•', '–', '—',
	'˜', '™', 'š', '›', 'œ', 0, 'ž', 'Ÿ',
}

// decodeWinAnsi converte bytes em WinAnsiEncoding para texto UTF-8. Fora da
// faixa 0x80–0x9F, a WinAnsiEncoding coincide com a Latin-1, em que o valor do
// byte já é o ponto de código.
func decodeWinAnsi(b []byte) string {
	var out strings.Builder
	out.Grow(len(b))
	for _, c := range b {
		switch {
		case c >= 0x80 && c <= 0x9F:
			if r := altoWinAnsi[c-0x80]; r != 0 {
				out.WriteRune(r)
			}
		default:
			out.WriteRune(rune(c))
		}
	}
	return out.String()
}
