package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/kre-college/lms/pkg/grading/repository/postgres"
	"github.com/kre-college/lms/pkg/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/stdlib"
	pg "github.com/kre-college/lms/pkg/db/postgres"
)

var testDBURL string
var db *pg.DB
var grade = models.Grade{
	ID:        1,
	Score:     2,
	CreatedAt: time.Now().Format(time.RFC3339),
	StudentID: 3,
	TeacherID: 4,
	EventID:   5,
	SubjectID: 6,
	IsDeleted: false,
}

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "",
		Env: []string{
			"POSTGRES_USER=dev",
			"POSTGRES_PASSWORD=12345",
			"POSTGRES_DB=grading",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		testDBURL = fmt.Sprintf("postgres://dev:12345@%v/grading?sslmode=disable", resource.GetHostPort("5432/tcp"))
		db, err = pg.NewDB(testDBURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()

	err = migrateScripts("../migrations", testDBURL)
	if err != nil {
		log.Print(err.Error())
		return
	}

	_ = resource.Expire(60)

	log.Printf("testDBURL: %s", testDBURL)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestInsertGrade(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       *models.Grade
		expectedErr error
	}{
		{
			name:  "OK",
			input: &grade,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			err := repo.InsertGrade(ctx, testCase.input)

			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestInsertGradeHistory(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       *models.Grade
		expectedErr error
	}{
		{
			name:  "OK",
			input: &grade,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			err := repo.InsertGradeHistory(ctx, testCase.input)

			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestFetchGrades(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		expect      []*models.Grade
		expectedErr error
	}{
		{
			name: "OK",
			expect: []*models.Grade{
				&grade,
			},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.FetchGrades(ctx)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestGetGradeByID(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       int
		expect      *models.Grade
		expectedErr error
	}{
		{
			name:   "OK",
			input:  grade.ID,
			expect: &grade,
		},
		{
			name:        "Not found",
			input:       55,
			expectedErr: sql.ErrNoRows,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.GetGradeByID(ctx, testCase.input)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestFetchGradesByStudentID(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       int
		expect      []*models.Grade
		expectedErr error
	}{
		{
			name:  "OK",
			input: grade.StudentID,
			expect: []*models.Grade{
				&grade,
			},
		},
		{
			name:   "Not found",
			input:  55,
			expect: []*models.Grade{},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.FetchGradesByStudentID(ctx, testCase.input)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestFetchGradesBySubjectID(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       int
		expect      []*models.Grade
		expectedErr error
	}{
		{
			name:  "OK",
			input: grade.SubjectID,
			expect: []*models.Grade{
				&grade,
			},
		},
		{
			name:   "Not found",
			input:  55,
			expect: []*models.Grade{},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.FetchGradesBySubjectID(ctx, testCase.input)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestFetchGradeHistory(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       int
		expect      []*models.Grade
		expectedErr error
	}{
		{
			name:  "OK",
			input: grade.ID,
			expect: []*models.Grade{
				&grade,
			},
		},
		{
			name:   "Not found",
			input:  55,
			expect: []*models.Grade{},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.FetchGradeHistory(ctx, testCase.input)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestDeleteGrade(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		input       int
		expectedErr error
	}{
		{
			name:  "OK",
			input: grade.ID,
		},
		{
			name:        "Not found",
			input:       55,
			expectedErr: sql.ErrNoRows,
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			err := repo.DeleteGrade(ctx, testCase.input)

			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func TestFetchGradesFromEmptyDatabase(t *testing.T) {
	repo := postgres.NewGradeRepository(db)
	ctx := context.Background()

	testTable := []struct {
		name        string
		expect      []*models.Grade
		expectedErr error
	}{
		{
			name:   "Empty fields",
			expect: []*models.Grade{},
		},
	}
	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := repo.FetchGrades(ctx)

			assert.Equal(t, testCase.expect, result)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}

func migrateScripts(migrations, dbURL string) error {
	mgrt, err := migrate.New("file://"+migrations, dbURL)
	if err != nil {
		return errors.Errorf("Could not open migration sources: %s, err: %s", migrations, err.Error())
	}

	err = mgrt.Up()
	if err != nil {
		return errors.Errorf("Could not apply migrations: %s, err: %s", migrations, err.Error())
	}
	return nil
}

