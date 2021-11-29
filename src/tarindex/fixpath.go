package tarindex

import (
	"os"
	"path"
	"strings"
)

type PathMod struct {
	BaseDir string
	ModDir  string
}

func (mod PathMod) FixPath(orig string) string {
	if strings.HasPrefix(orig, mod.BaseDir) {
		return strings.Replace(orig, mod.BaseDir, mod.ModDir, 1)
	}
	if orig == mod.BaseDir || orig+string(os.PathSeparator) == mod.BaseDir {
		return mod.ModDir
	}
	return path.Join(mod.ModDir, orig)
}
