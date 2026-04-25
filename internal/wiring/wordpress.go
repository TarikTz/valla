package wiring

import (
	"fmt"
	"strings"

	"github.com/tariktz/valla-cli/internal/registry"
)

const wordPressMySQLInternalPort = 3306

// GenerateWordPressEnv produces the content of the .env file for the WordPress preset.
func GenerateWordPressEnv(ctx registry.WeldContext) string {
	mysql := ctx.DBConfigs["mysql"]
	var builder strings.Builder
	fmt.Fprintf(&builder, "WORDPRESS_PORT=%d\n", ctx.FrontendPort)
	fmt.Fprintf(&builder, "MYSQL_PORT=%d\n", mysql.Port)
	fmt.Fprintf(&builder, "MYSQL_DATABASE=%s\n", mysql.Name)
	fmt.Fprintf(&builder, "MYSQL_USER=%s\n", mysql.User)
	fmt.Fprintf(&builder, "MYSQL_PASSWORD=%s\n", mysql.Password)
	fmt.Fprintf(&builder, "MYSQL_ROOT_PASSWORD=%s\n", mysql.Password+"_root")
  fmt.Fprintf(&builder, "WORDPRESS_DB_HOST=%s:%d\n", mysql.Host, wordPressMySQLInternalPort)
	fmt.Fprintf(&builder, "WORDPRESS_DB_NAME=%s\n", mysql.Name)
	fmt.Fprintf(&builder, "WORDPRESS_DB_USER=%s\n", mysql.User)
	fmt.Fprintf(&builder, "WORDPRESS_DB_PASSWORD=%s\n", mysql.Password)
	return builder.String()
}

// GenerateWordPressCompose produces the content of docker-compose.yml for the WordPress preset.
func GenerateWordPressCompose() string {
	return `version: "3.8"

services:
  wordpress:
    image: wordpress:6.5-php8.3-apache
    ports:
      - "${WORDPRESS_PORT}:80"
    environment:
      WORDPRESS_DB_HOST: ${WORDPRESS_DB_HOST}
      WORDPRESS_DB_USER: ${WORDPRESS_DB_USER}
      WORDPRESS_DB_PASSWORD: ${WORDPRESS_DB_PASSWORD}
      WORDPRESS_DB_NAME: ${WORDPRESS_DB_NAME}
    volumes:
      - ./wordpress:/var/www/html
    depends_on:
      - db

  db:
    image: mysql:8.4
    ports:
      - "${MYSQL_PORT}:3306"
    environment:
      MYSQL_DATABASE: ${MYSQL_DATABASE}
      MYSQL_USER: ${MYSQL_USER}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
    volumes:
      - mysql-data:/var/lib/mysql

volumes:
  mysql-data:
`
}
