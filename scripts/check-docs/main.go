// check-docs checks relative Markdown links and image references.
package main

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	bad := false
	re := regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == "bin" || d.Name() == ".git" || p == "demos/work" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".md" {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		for _, m := range re.FindAllSubmatch(b, -1) {
			target := strings.Trim(string(m[1]), "<>")
			if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target = strings.Split(target, "#")[0]
			target = strings.Split(target, "?")[0]
			if target == "" {
				continue
			}
			target, _ = url.PathUnescape(target)
			if _, e = os.Stat(filepath.Join(filepath.Dir(p), target)); e != nil {
				fmt.Fprintf(os.Stderr, "%s: missing %s\n", p, target)
				bad = true
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		bad = true
	}
	if bad {
		os.Exit(1)
	}
	fmt.Println("Relative Markdown links and images: OK")
}
