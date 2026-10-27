package migrationhandler

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/golang-migrate/migrate/v4"
)

// fakeEngine records the calls Migrator makes and returns the configured results.
type fakeEngine struct {
	version uint
	dirty   bool
	verErr  error
	upErr   error
	stepErr error
	forceEr error
	srcErr  error
	dbErr   error

	upCalls    int
	stepCalls  []int
	forceCalls []int
	closeCalls int
}

func (f *fakeEngine) Version() (uint, bool, error) { return f.version, f.dirty, f.verErr }
func (f *fakeEngine) Up() error                    { f.upCalls++; return f.upErr }
func (f *fakeEngine) Steps(n int) error            { f.stepCalls = append(f.stepCalls, n); return f.stepErr }
func (f *fakeEngine) Force(v int) error            { f.forceCalls = append(f.forceCalls, v); return f.forceEr }
func (f *fakeEngine) Close() (error, error)        { f.closeCalls++; return f.srcErr, f.dbErr }

func setFake(t *testing.T, e Engine, maxVersion uint, err error) {
	t.Helper()
	t.Cleanup(SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
		return e, maxVersion, err
	}))
}

func Test_SetEngineFunc_New(t *testing.T) {
	fake := &fakeEngine{}
	wantFS := fstest.MapFS{"000001_a.up.sql": {Data: []byte("SELECT 1;")}}

	var gotDSN, gotTable string
	var gotFS fs.FS
	t.Cleanup(SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (Engine, uint, error) {
		gotDSN, gotFS, gotTable = dsn, fsys, versionTable
		return fake, 7, nil
	}))

	m, err := New("dsn-a", wantFS, "tbl-a")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if m.MaxVersion() != 7 {
		t.Errorf("Wrong match. expect: 7, got: %d", m.MaxVersion())
	}
	if m.m != Engine(fake) {
		t.Errorf("Wrong match. expect: the fake engine, got: %v", m.m)
	}
	if gotDSN != "dsn-a" || gotTable != "tbl-a" || !reflect.DeepEqual(gotFS, fs.FS(wantFS)) {
		t.Errorf("Wrong match. expect: the arguments of New, got: %q, %q, %v", gotDSN, gotTable, gotFS)
	}
}

func Test_SetEngineFunc_error(t *testing.T) {
	errOpen := errors.New("open failed")
	setFake(t, nil, 0, errOpen)

	m, err := New("dsn", nil, "tbl")
	if !errors.Is(err, errOpen) || m != nil {
		t.Errorf("Wrong match. expect: nil migrator and the open error, got: %v, %v", m, err)
	}
}

func Test_SetEngineFunc_restore(t *testing.T) {
	var firstCalls, secondCalls int
	restoreFirst := SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
		firstCalls++
		return &fakeEngine{}, 1, nil
	})
	restoreSecond := SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
		secondCalls++
		return &fakeEngine{}, 2, nil
	})

	if _, err := New("dsn", nil, "tbl"); err != nil || secondCalls != 1 || firstCalls != 0 {
		t.Fatalf("Wrong match. expect: second factory used, got: first %d second %d err %v", firstCalls, secondCalls, err)
	}

	restoreSecond()
	if _, err := New("dsn", nil, "tbl"); err != nil || firstCalls != 1 {
		t.Fatalf("Wrong match. expect: first factory after restore, got: first %d err %v", firstCalls, err)
	}

	restoreFirst()
	// The default needs a real server: an unparsable dsn fails before any connection is tried.
	if _, err := New("not a dsn", nil, "tbl"); err == nil || firstCalls != 1 || secondCalls != 1 {
		t.Errorf("Wrong match. expect: the default factory error, got: first %d second %d err %v", firstCalls, secondCalls, err)
	}
}

func Test_SetEngineFunc_nilInstallsDefault(t *testing.T) {
	called := false
	restore := SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
		called = true
		return &fakeEngine{}, 1, nil
	})
	defer restore()

	restoreNil := SetEngineFunc(nil)
	if _, err := New("not a dsn", nil, "tbl"); err == nil || called {
		t.Errorf("Wrong match. expect: the default factory error, got: called %v err %v", called, err)
	}

	restoreNil()
	if _, err := New("dsn", nil, "tbl"); err != nil || !called {
		t.Errorf("Wrong match. expect: the fake factory again, got: called %v err %v", called, err)
	}
}

func Test_SetEngineFunc_reenter(t *testing.T) {
	var restoreInner func()
	restore := SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
		// A factory may call the setter and New without deadlock.
		restoreInner = SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
			return &fakeEngine{}, 9, nil
		})
		return &fakeEngine{}, 1, nil
	})
	defer restore()

	if _, err := New("dsn", nil, "tbl"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer restoreInner()

	m, err := New("dsn", nil, "tbl")
	if err != nil || m.MaxVersion() != 9 {
		t.Errorf("Wrong match. expect: max 9, got: %v, %v", m, err)
	}
}

func Test_SetEngineFunc_concurrent(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			restore := SetEngineFunc(func(string, fs.FS, string) (Engine, uint, error) {
				return &fakeEngine{}, 1, nil
			})
			restore()
		}
	}()

	for i := 0; i < 200; i++ {
		_, _ = New("not a dsn", nil, "tbl")
	}
	<-done
}

func Test_Run_fakeEngine(t *testing.T) {
	tests := []struct {
		name string
		fake *fakeEngine

		expectErr   bool
		expectUp    int
		expectClose int
	}{
		{"applies and closes", &fakeEngine{}, false, 1, 1},
		{"no change", &fakeEngine{upErr: migrate.ErrNoChange}, false, 1, 1},
		{"database ahead is skipped", &fakeEngine{version: 9}, false, 0, 1},
		{"dirty database is refused by the engine", &fakeEngine{version: 9, dirty: true, upErr: errors.New("dirty")}, true, 1, 1},
		{"version error", &fakeEngine{verErr: errors.New("boom")}, true, 0, 1},
		{"close errors are ignored by Run", &fakeEngine{srcErr: errors.New("a"), dbErr: errors.New("b")}, false, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFake(t, tt.fake, 3, nil)

			err := Run("dsn", nil, "tbl")
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.fake.upCalls != tt.expectUp || tt.fake.closeCalls != tt.expectClose {
				t.Errorf("Wrong match. expect up %d close %d, got: up %d close %d", tt.expectUp, tt.expectClose, tt.fake.upCalls, tt.fake.closeCalls)
			}
		})
	}
}

func Test_Run_openError(t *testing.T) {
	errOpen := errors.New("open failed")
	setFake(t, nil, 0, errOpen)

	if err := Run("dsn", nil, "tbl"); !errors.Is(err, errOpen) {
		t.Errorf("Wrong match. expect: %v, got: %v", errOpen, err)
	}
}

func Test_Migrator_fakeEngine(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"up with a nil version applies", func(t *testing.T) {
			fake := &fakeEngine{verErr: migrate.ErrNilVersion}
			m := &Migrator{m: fake, maxVersion: 2}
			if err := m.Up(); err != nil || fake.upCalls != 1 {
				t.Errorf("Wrong match. expect: ok and one up, got: %v, %d", err, fake.upCalls)
			}
		}},

		{"status of an unversioned database", func(t *testing.T) {
			m := &Migrator{m: &fakeEngine{verErr: migrate.ErrNilVersion}}
			v, dirty, err := m.Status()
			if v != 0 || dirty || err != nil {
				t.Errorf("Wrong match. expect: 0, false, nil, got: %d, %v, %v", v, dirty, err)
			}
		}},

		{"status returns an engine error", func(t *testing.T) {
			m := &Migrator{m: &fakeEngine{verErr: errors.New("boom")}}
			if _, _, err := m.Status(); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		}},

		{"down on an unversioned database does nothing", func(t *testing.T) {
			fake := &fakeEngine{verErr: migrate.ErrNilVersion}
			m := &Migrator{m: fake}
			if err := m.Down(); err != nil || len(fake.stepCalls) != 0 {
				t.Errorf("Wrong match. expect: ok and no step, got: %v, %v", err, fake.stepCalls)
			}
		}},

		{"down reverts one step", func(t *testing.T) {
			fake := &fakeEngine{version: 2}
			m := &Migrator{m: fake}
			if err := m.Down(); err != nil || len(fake.stepCalls) != 1 || fake.stepCalls[0] != -1 {
				t.Errorf("Wrong match. expect: ok and step -1, got: %v, %v", err, fake.stepCalls)
			}
		}},

		{"down returns the step error and ignores no change", func(t *testing.T) {
			m := &Migrator{m: &fakeEngine{version: 2, stepErr: errors.New("boom")}}
			if err := m.Down(); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}

			m = &Migrator{m: &fakeEngine{version: 2, stepErr: migrate.ErrNoChange}}
			if err := m.Down(); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		}},

		{"down returns the status error", func(t *testing.T) {
			m := &Migrator{m: &fakeEngine{verErr: errors.New("boom")}}
			if err := m.Down(); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		}},

		{"force passes the version, also one ahead of the image", func(t *testing.T) {
			fake := &fakeEngine{}
			m := &Migrator{m: fake, maxVersion: 2}
			if err := m.Force(5); err != nil || len(fake.forceCalls) != 1 || fake.forceCalls[0] != 5 {
				t.Errorf("Wrong match. expect: ok and force 5, got: %v, %v", err, fake.forceCalls)
			}
		}},

		{"force returns the engine error", func(t *testing.T) {
			m := &Migrator{m: &fakeEngine{forceEr: errors.New("boom")}, maxVersion: 2}
			if err := m.Force(1); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		}},

		{"close joins both errors", func(t *testing.T) {
			errSrc, errDB := errors.New("src"), errors.New("db")
			m := &Migrator{m: &fakeEngine{srcErr: errSrc, dbErr: errDB}}
			err := m.Close()
			if !errors.Is(err, errSrc) || !errors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: both errors, got: %v", err)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func Test_defaultEngine_errors(t *testing.T) {
	valid := fstest.MapFS{
		"000001_a.up.sql":   {Data: []byte("SELECT 1;")},
		"000001_a.down.sql": {Data: []byte("SELECT 1;")},
	}

	tests := []struct {
		name string
		dsn  string
		fsys fs.FS
	}{
		{"unparsable dsn", "not a dsn", valid},
		{"empty migration directory", "u:p@tcp(127.0.0.1:1)/db?timeout=1s", fstest.MapFS{}},
		{"unreachable database", "u:p@tcp(127.0.0.1:1)/db?timeout=1s", valid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, v, err := defaultEngine(tt.dsn, tt.fsys, "tbl")
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if e != nil || v != 0 {
				t.Errorf("Wrong match. expect: untyped nil engine and 0, got: %v, %d", e, v)
			}
		})
	}
}
