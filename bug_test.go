package bug

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"

	"entgo.io/bug/ent/post"
	"entgo.io/bug/ent/user"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"entgo.io/bug/ent"
	"entgo.io/bug/ent/enttest"
)

func TestBugSQLite(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	test(t, client)
}

func TestBugMySQL(t *testing.T) {
	for version, port := range map[string]int{"56": 3306, "57": 3307, "8": 3308} {
		addr := net.JoinHostPort("localhost", strconv.Itoa(port))
		t.Run(version, func(t *testing.T) {
			client := enttest.Open(t, dialect.MySQL, fmt.Sprintf("root:pass@tcp(%s)/test?parseTime=True", addr))
			defer client.Close()
			test(t, client)
		})
	}
}

func TestBugPostgres(t *testing.T) {
	for version, port := range map[string]int{"10": 5430, "11": 5431, "12": 5432, "13": 5433, "14": 5434} {
		t.Run(version, func(t *testing.T) {
			client := enttest.Open(t, dialect.Postgres, fmt.Sprintf("host=localhost port=%d user=postgres dbname=test password=pass sslmode=disable", port))
			defer client.Close()
			test(t, client)
		})
	}
}

func TestBugMaria(t *testing.T) {
	for version, port := range map[string]int{"10.5": 4306, "10.2": 4307, "10.3": 4308} {
		t.Run(version, func(t *testing.T) {
			addr := net.JoinHostPort("localhost", strconv.Itoa(port))
			client := enttest.Open(t, dialect.MySQL, fmt.Sprintf("root:pass@tcp(%s)/test?parseTime=True", addr))
			defer client.Close()
			test(t, client)
		})
	}
}

func test(t *testing.T, client *ent.Client) {
	ctx := context.Background()
	client.User.Delete().ExecX(ctx)

	a := client.User.Create().SetName("A").SaveX(ctx)
	b := client.User.Create().SetName("B").SaveX(ctx)
	c := client.User.Create().SetName("C").SaveX(ctx)

	createPost := func(u *ent.User, prefix string, number int) {
		for i := 0; i < number; i++ {
			client.Post.Create().
				SetName(fmt.Sprintf("POST: %s-%d", prefix, i)).
				SetCreator(u).
				ExecX(ctx)
		}
	}

	createPost(a, "a", 5)
	createPost(b, "b", 10)
	createPost(c, "c", 2)

	usersByPost := client.Debug().User.Query().Order(func(selector *sql.Selector) {
		// not very elegant, since we are changing more than the order for this case
		// but I'm not sure how order by count of another table without doing a join
		ptbl := sql.Table(post.Table)
		subQuery := sql.Select("COUNT(*) AS c", ptbl.C(post.FieldUserID)).
			From(ptbl).
			GroupBy(ptbl.C(post.FieldUserID)).
			As("ord")

		selector.Join(subQuery).On(selector.C(user.FieldID), subQuery.C(post.FieldUserID))
		selector.OrderBy(sql.Desc(subQuery.C("c")))
	}).AllX(ctx)

	require.Len(t, usersByPost, 3)
	require.Equal(t, b.ID, usersByPost[0].ID)
	require.Equal(t, a.ID, usersByPost[1].ID)
	require.Equal(t, c.ID, usersByPost[2].ID)
}
