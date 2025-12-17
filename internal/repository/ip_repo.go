package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/StellaShiina/ktauth/internal/model"
)

type IPRepo struct {
	db *sql.DB
}

func NewIPRepo(db *sql.DB) *IPRepo {
	return &IPRepo{db: db}
}

func (r *IPRepo) AddIP(ctx context.Context, cidr string, rule_type model.IPRuleType) (int64, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO ip (cidr, rule_type) VALUES (?, ?)", cidr, rule_type)
	if err != nil {
		return 0, fmt.Errorf("AddIP: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("AddIP: %v", err)
	}
	return id, nil
}

func (r *IPRepo) DelIP(ctx context.Context, cidr string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ip WHERE cidr = ?", cidr)
	if err != nil {
		return err
	}
	return nil
}

func (r *IPRepo) QueryIP(ctx context.Context, cidr string) (model.IPRuleType, error) {
	var rule_type model.IPRuleType

	row := r.db.QueryRowContext(ctx, "SELECT rule_type FROM ip WHERE cidr = ?", cidr)
	if err := row.Scan(&rule_type); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("No such IP: %s", cidr)
		}
		return "", fmt.Errorf("Error when scanning: %v", err)
	}
	return rule_type, nil
}

func (r *IPRepo) GetIPs(ctx context.Context) ([]model.IP, error) {
	var ips []model.IP

	rows, err := r.db.QueryContext(ctx, "SELECT cidr, rule_type FROM ip;")

	if err != nil {
		return nil, fmt.Errorf("GetIPs error %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ip model.IP
		// if err := rows.Scan(&ip.ID, &ip.CIDR, &ip.RuleType, &ip.Note, &ip.CreateAt, &ip.UpdateAt); err != nil {
		if err := rows.Scan(&ip.CIDR, &ip.RuleType); err != nil {
			return nil, fmt.Errorf("GetIPs error %v", err)
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetIPs error %v", err)
	}
	return ips, nil
}

func (r *IPRepo) GetIPsByType(ctx context.Context, rule_type model.IPRuleType) ([]string, error) {
	var ips []string

	rows, err := r.db.QueryContext(ctx, "SELECT cidr FROM ip WHERE rule_type = ?", rule_type)

	if err != nil {
		return nil, fmt.Errorf("Query error: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("Scan error: %v", err)
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows error: %v", err)
	}
	return ips, nil
}
