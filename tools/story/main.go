// Command story gera a imagem de story (1080x1920) de um post do blog,
// no estilo do tema hugo-paper.
//
//	go run ./tools/story                                  # post mais recente
//	go run ./tools/story -post content/posts/algo.md      # post específico
//	go run ./tools/story -summary "resumo que instiga"    # sobrescreve o resumo
package main

import (
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/*.ttf
var fonts embed.FS

// ---------------------------------------------------------------- estilo ---

const (
	width   = 1080
	height  = 1920
	pad     = 104  // margem lateral
	top     = 430  // início do conteúdo, fora da UI do Instagram
	footerY = 1620 // linha de base do rodapé

	titleMax   = 82 // corpo inicial do título (diminui até caber)
	titleMin   = 52
	titleLines = 6
)

type theme struct {
	name   string
	bg, fg color.RGBA
}

var themes = []theme{
	// linen: cor padrão do tema (#faf8f1) com texto preto
	{"linen", color.RGBA{250, 248, 241, 255}, color.RGBA{0, 0, 0, 255}},
	// dark: mesmo fundo com overlay black/85 e texto branco
	{"dark", color.RGBA{38, 37, 36, 255}, color.RGBA{255, 255, 255, 255}},
}

var meses = [...]string{
	"jan", "fev", "mar", "abr", "mai", "jun",
	"jul", "ago", "set", "out", "nov", "dez",
}

// ------------------------------------------------------------------ post ---

type post struct {
	Eyebrow string
	Title   string
	Date    time.Time
	Tag     string
	Summary string
	Footer  string
}

func (p post) meta() string {
	var parts []string
	if !p.Date.IsZero() {
		parts = append(parts, fmt.Sprintf("%d %s %d",
			p.Date.Day(), meses[p.Date.Month()-1], p.Date.Year()))
	}
	if p.Tag != "" {
		parts = append(parts, "#"+p.Tag)
	}
	return strings.Join(parts, "  ·  ")
}

// ----------------------------------------------------------------- fontes ---

func face(file string, size float64) font.Face {
	b, err := fonts.ReadFile("fonts/" + file)
	if err != nil {
		log.Fatalf("fonte %s: %v (baixe o Inter em github.com/rsms/inter)", file, err)
	}
	f, err := opentype.Parse(b)
	if err != nil {
		log.Fatalf("fonte %s: %v", file, err)
	}
	// DPI 72 faz 1 ponto valer exatamente 1 pixel
	fc, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("fonte %s: %v", file, err)
	}
	return fc
}

// ---------------------------------------------------------------- desenho ---

// mix simula opacidade (o equivalente a text-black/60 do tema).
func mix(fg, bg color.RGBA, alpha float64) color.RGBA {
	f := func(a, b uint8) uint8 {
		return uint8(float64(a)*alpha + float64(b)*(1-alpha) + 0.5)
	}
	return color.RGBA{f(fg.R, bg.R), f(fg.G, bg.G), f(fg.B, bg.B), 255}
}

type canvas struct {
	img *image.RGBA
	th  theme
}

func (c *canvas) text(x, y int, s string, fc font.Face, alpha float64) {
	d := &font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(mix(c.th.fg, c.th.bg, alpha)),
		Face: fc,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func (c *canvas) rule(x, y, w, h int, alpha float64) {
	r := image.Rect(x, y, x+w, y+h)
	draw.Draw(c.img, r, image.NewUniform(mix(c.th.fg, c.th.bg, alpha)), image.Point{}, draw.Src)
}

func measure(s string, fc font.Face) int {
	return font.MeasureString(fc, s).Round()
}

// wrap quebra o texto em linhas que caibam em maxW.
func wrap(s string, fc font.Face, maxW int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		test := word
		if line != "" {
			test = line + " " + word
		}
		if measure(test, fc) <= maxW {
			line = test
			continue
		}
		if line != "" {
			lines = append(lines, line)
		}
		line = word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// fitTitle diminui o corpo do título até caber em maxLines.
func fitTitle(s string, maxW int) (font.Face, []string, float64) {
	for size := float64(titleMax); size > titleMin; size -= 2 {
		fc := face("Inter-SemiBold.ttf", size)
		if lines := wrap(s, fc, maxW); len(lines) <= maxLines() {
			return fc, lines, size
		}
	}
	fc := face("Inter-SemiBold.ttf", titleMin)
	return fc, wrap(s, fc, maxW), titleMin
}

func maxLines() int { return titleLines }

// letterspace imita o tracking-wider do tema.
func letterspace(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func render(p post, th theme) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(th.bg), image.Point{}, draw.Src)
	c := &canvas{img: img, th: th}

	box := width - 2*pad
	y := top

	// sobrancelha
	eyebrow := face("Inter-Medium.ttf", 26)
	c.text(pad, y, letterspace(p.Eyebrow), eyebrow, 0.45)
	y += 74

	// título (h1 semibold)
	titleFace, lines, size := fitTitle(p.Title, box)
	lh := int(size * 1.18)
	for _, line := range lines {
		c.text(pad, y+int(size*0.78), line, titleFace, 1)
		y += lh
	}
	y += 30

	// meta (text-xs opacity-60)
	if m := p.meta(); m != "" {
		c.text(pad, y+24, m, face("Inter-Regular.ttf", 30), 0.55)
		y += 78
	}

	// filete
	c.rule(pad, y, 120, 2, 0.18)
	y += 66

	// resumo (article text-lg leading-[1.8])
	sumSize := 42.0
	sumFace := face("Inter-Regular.ttf", sumSize)
	for _, line := range wrap(p.Summary, sumFace, box) {
		c.text(pad, y+int(sumSize*0.78), line, sumFace, 0.78)
		y += int(sumSize * 1.62)
	}

	// rodapé
	c.rule(pad, footerY-52, box, 2, 0.10)
	c.text(pad, footerY+24, p.Footer, face("Inter-Medium.ttf", 30), 0.55)

	return img
}

// ----------------------------------------------------------- front matter ---

// parse lê title, date, tags e summary do front matter YAML de um post.
func parse(path string) (post, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return post{}, err
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	var fm, body string
	if strings.HasPrefix(text, "---\n") {
		if i := strings.Index(text[4:], "\n---"); i >= 0 {
			fm, body = text[4:4+i], text[4+i+4:]
		}
	}
	if fm == "" {
		return post{}, fmt.Errorf("%s: front matter YAML não encontrado", path)
	}

	p := post{}
	inTags := false
	for _, line := range strings.Split(fm, "\n") {
		key, val, ok := strings.Cut(line, ":")
		trimmed := strings.TrimSpace(line)

		// tags em lista de bloco (- espiritualidade)
		if inTags && strings.HasPrefix(trimmed, "- ") {
			if p.Tag == "" {
				p.Tag = unquote(strings.TrimPrefix(trimmed, "- "))
			}
			continue
		}
		inTags = false
		if !ok {
			continue
		}

		val = unquote(strings.TrimSpace(val))
		switch strings.TrimSpace(key) {
		case "title":
			p.Title = val
		case "summary", "description":
			p.Summary = val
		case "date":
			for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
				if t, err := time.Parse(layout, val); err == nil {
					p.Date = t
					break
				}
			}
		case "tags":
			if val == "" { // lista de bloco nas linhas seguintes
				inTags = true
				break
			}
			// lista inline: ['filosofia', 'x']
			val = strings.Trim(val, "[]")
			if first, _, _ := strings.Cut(val, ","); first != "" {
				p.Tag = unquote(strings.TrimSpace(first))
			}
		}
	}

	// sem summary no front matter: usa o primeiro parágrafo do corpo
	if p.Summary == "" {
		p.Summary = firstParagraph(body)
	}
	return p, nil
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	for _, q := range []string{`"`, `'`} {
		if len(s) >= 2 && strings.HasPrefix(s, q) && strings.HasSuffix(s, q) {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// stripMarkdown remove ênfases e links do texto usado como resumo.
var mdLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

func stripMarkdown(s string) string {
	s = mdLink.ReplaceAllString(s, "$1")
	return strings.NewReplacer("**", "", "*", "", "`", "", "_", "").Replace(s)
}

func firstParagraph(body string) string {
	for _, block := range strings.Split(strings.TrimSpace(body), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" || strings.HasPrefix(block, "#") || strings.HasPrefix(block, ">") {
			continue
		}
		block = strings.Join(strings.Fields(stripMarkdown(block)), " ")
		if len(block) > 220 { // corta no fim de frase mais próximo
			if i := strings.LastIndex(block[:220], ". "); i > 80 {
				return block[:i+1]
			}
			return strings.TrimSpace(block[:220]) + "…"
		}
		return block
	}
	return ""
}

// siteRoot sobe a árvore de diretórios até achar a raiz do site Hugo,
// para que o comando funcione de qualquer pasta do projeto.
func siteRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		for _, name := range []string{
			"hugo.toml", "hugo.yaml", "hugo.json",
			"config.toml", "config.yaml",
		} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// latest devolve o post mais recente (por data do front matter) em dir.
func latest(dir string) (string, error) {
	var newest string
	var newestAt time.Time
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), "_") || filepath.Base(path) == "index.md" {
			return nil
		}
		p, err := parse(path)
		if err != nil || p.Date.IsZero() {
			return nil
		}
		if p.Date.After(newestAt) {
			newest, newestAt = path, p.Date
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if newest == "" {
		return "", fmt.Errorf("nenhum post com data encontrado em %s", dir)
	}
	return newest, nil
}

// ------------------------------------------------------------------ main ---

func main() {
	var (
		postPath = flag.String("post", "", "caminho do post (padrão: o mais recente)")
		dir      = flag.String("dir", "", "diretório de conteúdo (padrão: <raiz>/content)")
		out      = flag.String("out", "", "diretório de saída (padrão: <raiz>/stories)")
		title    = flag.String("title", "", "título (sobrescreve o front matter)")
		summary  = flag.String("summary", "", "resumo (sobrescreve o front matter)")
		eyebrow  = flag.String("eyebrow", "NOVO NO BLOG", "texto da sobrancelha")
		footer   = flag.String("footer", "leia agora", "texto do rodapé")
	)
	flag.Parse()

	if *dir == "" || *out == "" {
		root := siteRoot()
		if *dir == "" {
			*dir = filepath.Join(root, "content")
		}
		if *out == "" {
			*out = filepath.Join(root, "stories")
		}
	}

	path := *postPath
	if path == "" {
		var err error
		if path, err = latest(*dir); err != nil {
			log.Fatal(err)
		}
	}

	p, err := parse(path)
	if err != nil {
		log.Fatal(err)
	}
	if *title != "" {
		p.Title = *title
	}
	if *summary != "" {
		p.Summary = *summary
	}
	p.Eyebrow, p.Footer = *eyebrow, *footer

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	slug := strings.TrimSuffix(filepath.Base(path), ".md")
	for _, th := range themes {
		file := filepath.Join(*out, fmt.Sprintf("%s-%s.png", slug, th.name))
		f, err := os.Create(file)
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, render(p, th)); err != nil {
			log.Fatal(err)
		}
		f.Close()
		fmt.Println(file)
	}
}
