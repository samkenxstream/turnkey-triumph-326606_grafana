package featuremgmt

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestFeatureToggleTypeScript(t *testing.T) {
	tsgen := generateTypeScript()

	fpath := "../../../packages/grafana-data/src/types/featureToggles.gen.ts"
	body, err := ioutil.ReadFile(fpath)
	if err == nil && tsgen != string(body) {
		err = fmt.Errorf("feature toggle typescript does not match")
	}

	if err != nil {
		_ = os.WriteFile(fpath, []byte(tsgen), 0644)
		t.Errorf("Feature toggle typescript does not match: %s", err.Error())
		t.Fail()
	}
}

func generateTypeScript() string {
	buf := `// NOTE: This file was auto generated.  DO NOT EDIT DIRECTLY!
// To change feature flags, edit:
//  pkg/services/featuremgmt/registry.go
// Then run tests in:
//  pkg/services/featuremgmt/toggles_gen_test.go

/**
 * Describes available feature toggles in Grafana. These can be configured via
 * conf/custom.ini to enable features under development or not yet available in
 * stable version.
 *
 * @public
 */
export interface FeatureToggles {
  // [name: string]?: boolean; // support any string value

`
	for _, flag := range standardFeatureFlags {
		buf += "  " + getTypeScriptKey(flag.Name) + "?: boolean;\n"
	}

	buf += "}\n"
	return buf
}

func getTypeScriptKey(key string) string {
	if strings.Contains(key, "-") || strings.Contains(key, ".") {
		return "['" + key + "']"
	}
	return key
}

func isLetterOrNumber(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c)
}

func asCamelCase(key string) string {
	parts := strings.FieldsFunc(key, isLetterOrNumber)
	for idx, part := range parts {
		parts[idx] = strings.Title(part)
	}
	return strings.Join(parts, "")
}

func TestGenerateToggleHelpers(t *testing.T) {
	tsgen, err := generateRegistry()
	require.NoError(t, err)

	fpath := "toggles_gen.go"
	body, err := ioutil.ReadFile(fpath)
	if err == nil {
		if diff := cmp.Diff(tsgen, string(body)); diff != "" {
			str := fmt.Sprintf("body mismatch (-want +got):\n%s\n", diff)
			err = fmt.Errorf(str)
		}
	}

	if err != nil {
		e2 := os.WriteFile(fpath, []byte(tsgen), 0644)
		if e2 != nil {
			t.Errorf("error writing file: %s", e2.Error())
		}
		abs, _ := filepath.Abs(fpath)
		t.Errorf("feature toggle helpers do not match: %s (%s)", err.Error(), abs)
		t.Fail()
	}
}

func generateRegistry() (string, error) {
	tmpl, err := template.New("fn").Parse(`
// {{.CamleCase}} checks for the flag: {{.Flag.Name}}{{.Ext}}
func (ft *FeatureToggles) Is{{.CamleCase}}Enabled() bool {
	return ft.manager.IsEnabled("{{.Flag.Name}}")
}
`)
	if err != nil {
		return "", err
	}

	data := struct {
		CamleCase string
		Flag      FeatureFlag
		Ext       string
	}{
		CamleCase: "?",
	}

	var buff bytes.Buffer

	buff.WriteString(`// NOTE: This file is autogenerated

package featuremgmt
`)

	for _, flag := range standardFeatureFlags {
		data.CamleCase = asCamelCase(flag.Name)
		data.Flag = flag
		data.Ext = ""

		if flag.Description != "" {
			data.Ext += "\n// " + flag.Description
		}

		if err := tmpl.Execute(&buff, data); err != nil {
			return buff.String(), err
		}
	}

	return buff.String(), nil
}
