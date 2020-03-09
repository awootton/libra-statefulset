package librastatefulset_test

import (
	"os"
	"testing"

	"github.com/awootton/knotfreeiot/librastatefulset"
)

func TestApply(t *testing.T) {

	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		librastatefulset.Apply(nil)
	}

}

func TestCreateConfigsLocally(t *testing.T) {

	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		librastatefulset.CreateConfigsLocally(nil)
	}

}
