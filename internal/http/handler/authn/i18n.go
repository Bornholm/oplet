package authn

import (
	"embed"

	"github.com/invopop/ctxi18n"
	"github.com/pkg/errors"
)

//go:embed i18n/*.yml
var i18n embed.FS

func init() {
	if err := ctxi18n.Load(i18n); err != nil {
		panic(errors.Wrap(err, "could not load translations"))
	}
}
