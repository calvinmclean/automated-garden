package migrate_test

import (
	"fmt"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/migrate"
	"github.com/stretchr/testify/require"
)

type A1 struct {
	Val int
}

func (A1) GetVersion() uint {
	return 1
}

type A2 struct {
	Val int
}

func (A2) GetVersion() uint {
	return 2
}

type A3 struct {
	Val int
}

func (A3) GetVersion() uint {
	return 3
}

type A4 struct {
	Val int
}

func (A4) GetVersion() uint {
	return 4
}

func TestMigrationWithVersionedStructs(t *testing.T) {
	migrations := []migrate.Migration{
		nil, // placeholder since I'm starting at V1
		migrate.NewMigration(
			"V1toV2",
			func(a1 A1) (A2, error) {
				return A2{
					Val: a1.Val + 1,
				}, nil
			},
		),
		migrate.NewMigration(
			"V2toV3",
			func(a2 A2) (A3, error) {
				return A3{
					Val: a2.Val + 1,
				}, nil
			},
		),
	}

	from := []A1{{1}}

	t.Run("MigrateV4NotFound", func(t *testing.T) {
		for _, err := range migrate.Each[A1, A4](migrations, from) {
			require.Error(t, err)
			require.Equal(t, "error running migration \"Unknown\"/3: migration not found", err.Error())
			require.ErrorIs(t, err, migrate.ErrNotFound)
		}
	})

	t.Run("MigrateEachV1toV3", func(t *testing.T) {
		for item, err := range migrate.Each[A1, A3](migrations, from) {
			require.NoError(t, err)
			require.Equal(t, 3, item.Val)
		}
	})

	t.Run("MigrateAll", func(t *testing.T) {
		out, err := migrate.All[A1, A3](migrations, from)
		require.NoError(t, err)
		require.Equal(t, []A3{{3}}, out)
	})
}

func TestMigrationErrors(t *testing.T) {
	migrations := []migrate.Migration{
		nil, // placeholder since I'm starting at V1
		migrate.NewMigration(
			"V1toV2",
			func(a1 A1) (A2, error) {
				return A2{
					Val: a1.Val + 1,
				}, nil
			},
		),
		migrate.NewMigration(
			"V2toV3Error",
			func(a2 A2) (A3, error) {
				return A3{}, fmt.Errorf("fatal error")
			},
		),
	}

	from := []A1{{1}}

	t.Run("MigrateV2toV3HandleErrors", func(t *testing.T) {
		for _, err := range migrate.Each[A1, A3](migrations, from) {
			require.Error(t, err)
			require.Equal(t, "error running migration \"V2toV3Error\"/2: fatal error", err.Error())
		}
	})
}

type B1 struct {
	Version uint
	Val     int
}

func (b *B1) GetVersion() uint {
	return b.Version
}

func (b *B1) SetVersion(v uint) {
	b.Version = v
}

type B2 struct {
	Version uint
	Val     int
}

func (b *B2) GetVersion() uint {
	return b.Version
}

func (b *B2) SetVersion(v uint) {
	b.Version = v
}

type B3 struct {
	Version uint
	Val     int
}

func (b *B3) GetVersion() uint {
	return b.Version
}

func (b *B3) SetVersion(v uint) {
	b.Version = v
}

// When Version is struct value and not a hard-coded method, it is auto-incremented on migrate and should
// error if the struct doesn't match the expected version
func TestMigrateWithValueVersion(t *testing.T) {
	migrations := []migrate.Migration{
		nil, // placeholder since I'm starting at V1
		migrate.NewMigration(
			"V1toV2",
			func(b1 *B1) (*B2, error) {
				return &B2{
					Val: b1.Val + 1,
				}, nil
			},
		),
		migrate.NewMigration(
			"V2toV3",
			func(b2 *B2) (*B3, error) {
				return &B3{
					Val: b2.Val + 1,
				}, nil
			},
		),
	}

	from := []*B1{{1, 1}}

	t.Run("MigrateEachV1toV3", func(t *testing.T) {
		for item, err := range migrate.Each[*B1, *B3](migrations, from) {
			require.NoError(t, err)
			require.Equal(t, 3, item.Val)
			require.Equal(t, uint(3), item.Version)
		}
	})

	t.Run("MigrateAll", func(t *testing.T) {
		out, err := migrate.All[*B1, *B3](migrations, from)
		require.NoError(t, err)
		require.Equal(t, []*B3{{3, 3}}, out)
	})

	// When an incorrect/unexpected version is set on the struct, migration fails
	t.Run("MigrateVersionIncorrect", func(t *testing.T) {
		in := []*B1{{2, 0}}
		out, err := migrate.All[*B1, *B3](migrations, in)
		require.Error(t, err)
		require.Equal(t, "error running migration \"V2toV3\"/2: unexpected From type", err.Error())
		require.ErrorIs(t, err, migrate.ErrInvalidFromType)
		require.Nil(t, out)
	})
}
