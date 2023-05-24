package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/lib/pq"
)

var sqlQuery = `
select p.name project_name, r.name repository, a.digest, t.name tagName, t.push_time
from project p,
     repository r,
     artifact a,
     tag t
where t.push_time < now() - interval '? week'
  and t.repository_id = r.repository_id
  and r.project_id = p.project_id
  and a.id = t.artifact_id
  `

type Tag struct {
	ProjectName string
	Repository  string
	Digest      string
	TagName     string
	PushTime    string
}

func main() {
	// Command line flags
	var (
		dbHost       = flag.String("db_host", "localhost", "Postgres database host")
		dbPort       = flag.Int("db_port", 5432, "Postgres database port")
		dbUser       = flag.String("db_user", "postgres", "Postgres database user")
		dbPass       = flag.String("db_pass", "root123", "Postgres database password")
		dbName       = flag.String("db_name", "registry", "Postgres database name")
		harborHost   = flag.String("harbor_host", "10.202.250.197", "Harbor host")
		harborUser   = flag.String("harbor_user", "admin", "Harbor user")
		harborPass   = flag.String("harbor_pass", "Harbor12345", "Harbor password")
		sqlCondition = flag.String("sql_condition", "", "SQL condition, empty or like '-sql_condition=\"p.name = 'tkg%' and r.name like 'tkg/sandbox/%'\"")
		dryRun       = flag.Bool("dry_run", false, "Whether to skip deleting files")
		weeks        = flag.Int("weeks", 1, "How many weeks to keep")
	)
	flag.Parse()

	// Validate command line arguments
	if *dbUser == "" || *dbName == "" {
		log.Fatalf("db_user, db_name are required")
	}

	// Connect to database
	db, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPass, *dbName))
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}
	sqlQuery = fmt.Sprintf(sqlQuery, *weeks)
	defer db.Close()
	if len(*sqlCondition) > 0 {
		sqlQuery = sqlQuery + " and " + *sqlCondition
	}
	rows, err := db.Query(sqlQuery)
	log.Printf("sqlQuery: %s", sqlQuery)
	if err != nil {
		log.Fatalf("failed to query database: %s", err)
	}
	defer rows.Close()
	tagList := []*Tag{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ProjectName, &tag.Repository, &tag.Digest, &tag.TagName, &tag.PushTime); err != nil {
			log.Fatalf("failed to scan database row: %s", err)
		}
		if strings.HasPrefix(tag.Repository, tag.ProjectName+"/") {
			tag.Repository = strings.TrimPrefix(tag.Repository, tag.ProjectName+"/")
		}
		tagList = append(tagList, &tag)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("failed to iterate database rows: %s", err)
	}

	for i, tag := range tagList {
		log.Printf("project: %s, repository: %s, digest: %s, tag: %s, push_time: %s", tag.ProjectName, tag.Repository, tag.Digest, tag.TagName, tag.PushTime)
		if !*dryRun {
			if err := deleteArtifact(tag, *harborHost, *harborUser, *harborPass); err != nil {
				log.Printf("failed to delete artifact: %s", err)
			}
			log.Printf("total %v, current %v\n", len(tagList), i+1)
		}
	}

	// Print summary
	if *dryRun {
		log.Printf("total artifact to delete: %d", len(tagList))
	} else {
		log.Printf("deleted %d artifacts", len(tagList))
	}
}

func deleteArtifact(tag *Tag, hostname string, username string, password string) error {

	// Construct the URL for deleting the tag
	url := fmt.Sprintf("https://%s/api/v2.0/projects/%s/repositories/%s/artifacts/%s/tags/%s", hostname, tag.ProjectName, tag.Repository, tag.Digest, tag.TagName)
	log.Printf("Deleting tag from url: %s", url)
	// Create a new HTTP request with DELETE method
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set basic authentication header
	req.SetBasicAuth(username, password)

	// Send the HTTP request
	client := &http.Client{}
	// skip TLS verify
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode == http.StatusOK {
		fmt.Println("tag deleted successfully.")
	} else {
		return fmt.Errorf("failed to delete tag. Status code: %d", resp.StatusCode)
	}
	return nil
}
