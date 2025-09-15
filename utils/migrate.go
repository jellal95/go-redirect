package utils

//import (
//	"fmt"
//	"log"
//)
//
//func Migrate() error {
//	if DB == nil {
//		return fmt.Errorf("database not initialized")
//	}
//
//	query := `
//	CREATE TABLE IF NOT EXISTS logs (
//		id SERIAL PRIMARY KEY,
//		type TEXT,
//		timestamp TIMESTAMP,
//		product_name TEXT,
//		url TEXT,
//		ip TEXT,
//		user_agent TEXT,
//		browser TEXT,
//		os TEXT,
//		device TEXT,
//		referer TEXT,
//		query_raw TEXT,
//		query_params JSONB,
//		headers JSONB,
//		geo JSONB
//	);
//	`
//
//	if err := DB.Exec(query).Error; err != nil {
//		return err
//	}
//
//	log.Println("Migration applied successfully ✅")
//	return nil
//}
