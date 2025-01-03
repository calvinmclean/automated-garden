package migrate

import (
	"iter"
)

type Versioned interface {
	GetVersion() uint
}

type IncrementVersion interface {
	SetVersion(uint)
}

type Migration interface {
	Migrate(Versioned) (any, error)
	Name() string
}

type Func[From, To Versioned] func(From) (To, error)

type migration[From, To Versioned] struct {
	name    string
	migrate Func[From, To]
}

func (m *migration[From, To]) Name() string {
	return m.name
}

func (m *migration[From, To]) Migrate(from Versioned) (any, error) {
	f, ok := from.(From)
	if !ok {
		return nil, ErrInvalidFromType
	}

	return m.migrate(f)
}

func NewMigration[From, To Versioned](name string, migrate Func[From, To]) Migration {
	return &migration[From, To]{
		name:    name,
		migrate: migrate,
	}
}

func All[From, To Versioned](migrations []Migration, from []From) ([]To, error) {
	result := []To{}

	for _, f := range from {
		to, err := migrateToFinalVersion[From, To](migrations, f)
		if err != nil {
			return nil, err
		}

		result = append(result, to)
	}

	return result, nil
}

func Each[From, To Versioned](migrations []Migration, from []From) iter.Seq2[To, error] {
	return func(yield func(To, error) bool) {
		for _, f := range from {
			to, err := migrateToFinalVersion[From, To](migrations, f)
			shouldContinue := yield(to, err)
			if !shouldContinue {
				return
			}
		}
	}
}

func runMigration[From, To Versioned](migrations []Migration, from From) (To, error) {
	v := from.GetVersion()

	if v >= uint(len(migrations)) {
		return *new(To), errNotFound(v)
	}
	m := migrations[v]

	to, err := m.Migrate(from)
	if err != nil {
		return *new(To), Error{err, m.Name(), v}
	}

	if versionSetter, ok := to.(IncrementVersion); ok {
		versionSetter.SetVersion(v + 1)
	}

	out, ok := to.(To)
	if !ok {
		return *new(To), Error{ErrInvalidToType, m.Name(), from.GetVersion()}
	}

	return out, err
}

func migrateToFinalVersion[From, To Versioned](migrations []Migration, from From) (To, error) {
	var next Versioned
	var err error

	next = from
	for {
		next, err = runMigration[Versioned, Versioned](migrations, next)
		if err != nil {
			return *new(To), err
		}

		if next.GetVersion() < uint(len(migrations)) {
			continue
		}

		result, ok := next.(To)
		if ok {
			return result, nil
		}
	}
}
