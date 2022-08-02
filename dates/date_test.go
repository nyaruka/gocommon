package dates_test

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nyaruka/gocommon/dates"
	"github.com/stretchr/testify/assert"
)

func TestDate(t *testing.T) {
	d1 := dates.NewDate(2019, 2, 20)

	assert.Equal(t, d1.Year, 2019)
	assert.Equal(t, d1.Month, time.Month(2))
	assert.Equal(t, d1.Day, 20)
	assert.Equal(t, d1.Weekday(), time.Weekday(3))
	assert.Equal(t, d1.WeekNum(), 8)
	assert.Equal(t, d1.YearDay(), 51)
	assert.Equal(t, "2019-02-20", d1.String())

	s, err := d1.Format("EEE, DD/MM/YYYY", "")
	assert.NoError(t, err)
	assert.Equal(t, "Wed, 20/02/2019", s)

	s, err = d1.Format("EEE, DD/MM/YYYY", "es-EC")
	assert.NoError(t, err)
	assert.Equal(t, "mié, 20/02/2019", s)

	_, err = d1.Format("ss:mm", "")
	assert.EqualError(t, err, "'ss' is not valid in a date formatting layout")

	d2 := dates.NewDate(2020, 1, 1)

	assert.Equal(t, d2.Year, 2020)
	assert.Equal(t, d2.Month, time.Month(1))
	assert.Equal(t, d2.Day, 1)
	assert.Equal(t, "2020-01-01", d2.String())

	// differs from d1 by 1 day
	d3 := dates.NewDate(2019, 2, 19)

	assert.Equal(t, d3.Year, 2019)
	assert.Equal(t, d3.Month, time.Month(2))
	assert.Equal(t, d3.Day, 19)
	assert.Equal(t, "2019-02-19", d3.String())

	// should be same date value as d1
	d4 := dates.ExtractDate(time.Date(2019, 2, 20, 9, 38, 30, 123456789, time.UTC))

	assert.Equal(t, d4.Year, 2019)
	assert.Equal(t, d4.Month, time.Month(2))
	assert.Equal(t, d4.Day, 20)
	assert.Equal(t, "2019-02-20", d4.String())

	assert.False(t, d1.Equal(d2))
	assert.False(t, d2.Equal(d1))
	assert.False(t, d1.Equal(d3))
	assert.False(t, d3.Equal(d1))
	assert.True(t, d1.Equal(d4))
	assert.True(t, d4.Equal(d1))

	assert.True(t, d1.Compare(d2) < 0)
	assert.True(t, d2.Compare(d1) > 0)
	assert.True(t, d1.Compare(d3) > 0)
	assert.True(t, d3.Compare(d1) < 0)
	assert.True(t, d1.Compare(d4) == 0)
	assert.True(t, d4.Compare(d1) == 0)

	kgl, _ := time.LoadLocation("Africa/Kigali")
	assert.Equal(t, time.Date(2020, 1, 2, 3, 4, 5, 6, time.UTC), dates.NewDate(2020, 1, 2).Combine(dates.NewTimeOfDay(3, 4, 5, 6), time.UTC))
	assert.Equal(t, time.Date(2020, 1, 2, 3, 4, 5, 6, kgl), dates.NewDate(2020, 1, 2).Combine(dates.NewTimeOfDay(3, 4, 5, 6), kgl))
}

func TestDateCalendarMethods(t *testing.T) {
	tests := []struct {
		date    dates.Date
		weekday time.Weekday
		weekNum int
		yearDay int
	}{
		// checked against Excel...
		{dates.NewDate(1981, 5, 28), time.Thursday, 22, 148},
		{dates.NewDate(2018, 12, 31), time.Monday, 53, 365},
		{dates.NewDate(2019, 1, 1), time.Tuesday, 1, 1},
		{dates.NewDate(2019, 7, 24), time.Wednesday, 30, 205},
		{dates.NewDate(2019, 12, 31), time.Tuesday, 53, 365},
		{dates.NewDate(2028, 1, 1), time.Saturday, 1, 1}, // leap year that starts on last day of week
		{dates.NewDate(2028, 12, 31), time.Sunday, 54, 366},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.weekday, tc.date.Weekday(), "incorrect week day for %s", tc.date)
		assert.Equal(t, tc.weekNum, tc.date.WeekNum(), "incorrect week num for %s", tc.date)
		assert.Equal(t, tc.yearDay, tc.date.YearDay(), "incorrect year day for %s", tc.date)
	}
}

func TestDateDBSerialization(t *testing.T) {
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	type foo struct {
		ID   int        `db:"id"`
		Name string     `db:"name"`
		Day  dates.Date `db:"day"`
	}

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10), day DATE)`)
	db.MustExec(`INSERT INTO foo (name, day) VALUES($1, $2)`, "Ann", dates.NewDate(2021, 3, 17))
	_, err := db.NamedExec(`INSERT INTO foo (name, day) VALUES(:name, :day)`, &foo{Name: "Bob", Day: dates.NewDate(2022, 5, 9)})
	assert.NoError(t, err)

	f := &foo{}
	err = db.Get(f, `SELECT id, name, day FROM foo WHERE id = 1`)
	assert.NoError(t, err)
	assert.Equal(t, dates.NewDate(2021, 3, 17), f.Day)

	err = db.Get(f, `SELECT id, name, day FROM foo WHERE id = 2`)
	assert.NoError(t, err)
	assert.Equal(t, dates.NewDate(2022, 5, 9), f.Day)
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("postgres", "postgres://nyaruka:nyaruka@localhost/gocommon_test?sslmode=disable&Timezone=UTC")
}
