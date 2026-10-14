package hints

import "testing"

func TestForFile(t *testing.T) {
	cases := map[string]Action{
		`C:\pagefile.sys`:                       System,
		`C:\hiberfil.sys`:                       System,
		`C:\dump.dmp`:                           Delete,
		`C:\Users\me\java_error_in_clion.hprof`: Delete,
		`C:\Users\me\AppData\Local\Temp\claude\x\tasks\a.output`:                              Delete,
		`C:\Users\me\Downloads\film.crdownload`:                                               Delete,
		`C:\Users\me\Downloads\setup.exe`:                                                     Delete,
		`C:\Users\me\Downloads\game.zip`:                                                      Delete,
		`C:\Users\me\Downloads\movie.avi`:                                                     Move,
		`C:\Users\me\.ollama\models\blobs\sha256-abc`:                                         Careful,
		`C:\Users\me\AppData\Local\Ollama\updates_v2\abc\OllamaSetup.exe`:                     Delete,
		`C:\ProgramData\Epic\EpicGamesLauncher\VaultCache\Lyra\data\Compressed.ddp`:           Cache,
		`C:\Users\me\AppData\Local\Google\Chrome\User Data\Default\Cache\Cache_Data\data_3`:   Cache,
		`C:\Users\me\AppData\Local\Google\Chrome\User Data\Default\File System\004\p\00\0003`: Careful,
		`C:\Users\me\AppData\Roaming\Cursor\User\globalStorage\state.vscdb`:                   Careful,
		`/home/me/.cache/go-build/ab/cd`:                                                      Cache,
	}

	for path, want := range cases {
		h, ok := ForFile(path)
		if !ok {
			t.Errorf("ForFile(%q): no hint, want %s", path, want.Label())
			continue
		}
		if h.Action != want {
			t.Errorf("ForFile(%q) = %s (%s), want %s", path, h.Action.Label(), h.Note, want.Label())
		}
	}

	for _, path := range []string{`C:\Users\me\Documents\thesis.docx`, `D:\Projects\app\main.go`} {
		if h, ok := ForFile(path); ok {
			t.Errorf("ForFile(%q) = %+v, want no hint", path, h)
		}
	}
}

func TestForDir(t *testing.T) {
	cases := map[string]Action{
		`C:\Users\me\AppData\Local\Temp`:           Delete,
		`C:\$Recycle.Bin`:                          Delete,
		`C:\Windows.old`:                           Delete,
		`C:\Windows\SoftwareDistribution\Download`: Cache,
		`D:\Game\Saved\DerivedDataCache`:           Cache,
		`C:\Users\me\.gradle\caches`:               Cache,
		`D:\web\app\node_modules`:                  Cache,
	}

	for path, want := range cases {
		h, ok := ForDir(path)
		if !ok || h.Action != want {
			t.Errorf("ForDir(%q) = %+v (ok=%v), want %s", path, h, ok, want.Label())
		}
	}

	// A folder named like an extension rule must not match file rules.
	if _, ok := ForDir(`D:\backups\old.zip`); ok {
		t.Error("ForDir matched a file-extension rule")
	}
}
