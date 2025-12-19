package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/StellaShiina/ktauth/internal/model"
)

type IPRepo struct {
	db *sql.DB
}

func NewIPRepo(db *sql.DB) *IPRepo {
	return &IPRepo{db: db}
}

func (r *IPRepo) AddIP(ctx context.Context, version model.IPVersion, ip_bin []byte, rule_type model.IPRuleType) error {
	result, err := r.db.ExecContext(ctx, "INSERT INTO ip (version, ip_bin, rule_type) VALUES (?, ?, ?)", version, ip_bin, rule_type)
	if err != nil {
		slog.Error("IPRepo AddIP: " + err.Error())
		return fmt.Errorf("IPRepo AddIP: %v", err)
	}
	_, err = result.LastInsertId()
	if err != nil {
		slog.Error("IPRepo AddIP: " + err.Error())
		return fmt.Errorf("IPRepo AddIP: %v", err)
	}
	slog.Debug("IPRepo AddIP success")
	return nil
}

func (r *IPRepo) DelIP(ctx context.Context, version model.IPVersion, ip_bin []byte) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM ip WHERE version = ? AND ip_bin = ?", version, ip_bin)
	if err != nil {
		slog.Error("IPRepo DelIP: " + err.Error())
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.Error("IPRepo DelIP: " + err.Error())
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("No such row!")
	}
	slog.Debug("IPRepo DelIP success")
	return nil
}

func (r *IPRepo) QueryIP(ctx context.Context, version model.IPVersion, ip_bin []byte) (model.IPRuleType, error) {
	var rule_type model.IPRuleType

	row := r.db.QueryRowContext(ctx, "SELECT rule_type FROM ip WHERE version = ? AND ip_bin = ?", version, ip_bin)
	if err := row.Scan(&rule_type); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("No such IP")
		}
		return "", fmt.Errorf("Error when scanning: %v", err)
	}
	return rule_type, nil
}

func (r *IPRepo) GetIPs(ctx context.Context) ([]model.IP, error) {
	var ips []model.IP

	rows, err := r.db.QueryContext(ctx, "SELECT * FROM ip;")

	if err != nil {
		slog.Error("GetIPs error: " + err.Error())
		return nil, fmt.Errorf("GetIPs error %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ip model.IP
		// if err := rows.Scan(&ip.ID, &ip.CIDR, &ip.RuleType, &ip.Note, &ip.CreateAt, &ip.UpdateAt); err != nil {
		if err := rows.Scan(&ip.ID, &ip.Version, &ip.IP_bin, &ip.RuleType, &ip.CreateAt, &ip.UpdateAt, &ip.Note); err != nil {
			slog.Error("GetIPs error: " + err.Error())
			return nil, fmt.Errorf("GetIPs error %v", err)
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		slog.Error("GetIPs error: " + err.Error())
		return nil, fmt.Errorf("GetIPs error %v", err)
	}
	return ips, nil
}
