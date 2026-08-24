package accounting

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/gobcn/radius-director/internal/generator"
)

// OpenMySQL opens and verifies a MySQL connection for one generated tenant.
func OpenMySQL(ctx context.Context, configuration generator.SQL) (*sql.DB, error) {
	address := net.JoinHostPort(configuration.Host, strconv.Itoa(configuration.Port))

	config := mysql.NewConfig()
	config.User = configuration.Username
	config.Passwd = configuration.Password
	config.Net = "tcp"
	config.Addr = address
	config.DBName = configuration.Database

	connector, err := mysql.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("create MySQL connector: %w", err)
	}

	database := sql.OpenDB(connector)
	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("connect to MySQL database: %w", err)
	}

	return database, nil
}
