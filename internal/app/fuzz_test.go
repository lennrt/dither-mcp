package app

import "testing"

func FuzzStrictJSON(f *testing.F) {
	for _, s := range []string{`{}`, `{"input":"image.png","output":"out.png"}`, `{"options":{"gamma":0}}`, `null`, `{"input":5}`, `{} {}`, `{"input":"../x"}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1<<20 {
			t.Skip()
		}
		var q RenderRequest
		_ = StrictJSON(b, &q)
	})
}
