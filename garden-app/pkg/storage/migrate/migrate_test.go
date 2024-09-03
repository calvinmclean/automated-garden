package migrate_test

import (
	"fmt"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/migrate"
	"github.com/stretchr/testify/require"
)

type A1 struct {
	Version uint
	Val     int
}

func (*A1) GetVersion() uint {
	return 1
}

func (a *A1) SetVersion(v uint) {
	a.Version = 1
}

type A2 struct {
	Version uint
	Val     int
}

func (*A2) GetVersion() uint {
	return 2
}

func (a *A2) SetVersion(v uint) {
	a.Version = 2
}

type A3 struct {
	Version uint
	Val     int
}

func (*A3) GetVersion() uint {
	return 3
}

func (a *A3) SetVersion(v uint) {
	a.Version = 3
}

type A4 struct {
	Version uint
	Val     int
}

func (*A4) GetVersion() uint {
	return 4
}

func (a *A4) SetVersion(v uint) {
	a.Version = 4
}

func TestMigration(t *testing.T) {
	migrations := []migrate.Migration{
		nil, // placeholder since I'm starting at V1
		migrate.NewMigration(
			"V1toV2",
			func(a1 *A1) (*A2, error) {
				return &A2{
					Val: a1.Val + 1,
				}, nil
			},
		),
		migrate.NewMigration(
			"V2toV3",
			func(a2 *A2) (*A3, error) {
				return &A3{
					Val: a2.Val + 1,
				}, nil
			},
		),
	}

	from := []*A1{{1, 1}}

	t.Run("MigrateV4NotFound", func(t *testing.T) {
		for _, err := range migrate.Each[*A1, *A4](migrations, from) {
			require.Error(t, err)
			require.Equal(t, "error running migration \"Unknown\"/3: migration not found", err.Error())
			require.ErrorIs(t, err, migrate.ErrNotFound)
		}
	})

	t.Run("MigrateEachV1toV3", func(t *testing.T) {
		for item, err := range migrate.Each[*A1, *A3](migrations, from) {
			require.NoError(t, err)
			require.Equal(t, 3, item.Val)
		}
	})

	t.Run("MigrateAll", func(t *testing.T) {
		out, err := migrate.All[*A1, *A3](migrations, from)
		require.NoError(t, err)
		require.Equal(t, []*A3{{3, 3}}, out)
	})
}

func TestMigrationErrors(t *testing.T) {
	migrations := []migrate.Migration{
		nil, // placeholder since I'm starting at V1
		migrate.NewMigration(
			"V1toV2",
			func(a1 *A1) (*A2, error) {
				return &A2{
					Val: a1.Val + 1,
				}, nil
			},
		),
		migrate.NewMigration(
			"V2toV3Error",
			func(a2 *A2) (*A3, error) {
				return nil, fmt.Errorf("fatal error")
			},
		),
	}

	from := []*A1{{1, 1}}

	t.Run("MigrateV2toV3HandleErrors", func(t *testing.T) {
		for _, err := range migrate.Each[*A1, *A3](migrations, from) {
			require.Error(t, err)
			require.Equal(t, "error running migration \"V2toV3Error\"/2: fatal error", err.Error())
		}
	})
}
