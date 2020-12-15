package test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var datasourceName = fmt.Sprintf(
	"%s:%s@tcp(%s)/%s?charset=utf8",
	"root",
	"secret",
	"127.0.0.1",
	"users_db",
)

func insertRecord(b *testing.B, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, "INSERT INTO isbns VALUES (CONNECTION_ID())")
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkMaxOpenConns1(b *testing.B) {

	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func TestConns1(t *testing.T) {

	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		t.Fatal(err)
	}
	//db.SetMaxOpenConns(10)
	//db.SetMaxIdleConns(1)
	defer db.Close()

	func() {
		fmt.Println('.')
		db.Ping()
		time.Sleep(time.Second * 2)
		fmt.Println('.')
		db.Ping()
		time.Sleep(time.Second * 2)
		fmt.Println('.')
		db.Ping()
	}()

	time.Sleep(time.Hour)

}

func BenchmarkMaxOpenConns2(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(2)
	defer db.Close()

	//b.RunParallel(func(pb *testing.PB) {
	//	for pb.Next() {
	//		insertRecord(b, db)
	//	}
	//})
}

func BenchmarkMaxOpenConns5(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(5)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxOpenConns10(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxOpenConnsUnlimited(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxIdleConnsNone(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxIdleConns(0)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxIdleConns1(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxIdleConns(1)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxIdleConns2(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxIdleConns(2)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxIdleConns5(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxIdleConns(5)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkMaxIdleConns10(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetMaxIdleConns(10)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkConnMaxLifetimeUnlimited(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetConnMaxLifetime(0)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkConnMaxLifetime1000(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetConnMaxLifetime(1000 * time.Millisecond)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkConnMaxLifetime500(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetConnMaxLifetime(500 * time.Millisecond)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkConnMaxLifetime200(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetConnMaxLifetime(200 * time.Millisecond)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}

func BenchmarkConnMaxLifetime100(b *testing.B) {
	db, err := sql.Open("mysql", datasourceName)
	if err != nil {
		b.Fatal(err)
	}
	db.SetConnMaxLifetime(100 * time.Millisecond)
	defer db.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			insertRecord(b, db)
		}
	})
}
