package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrIPNotFound = errors.New("ip rule not found")
var ErrIPExist = errors.New("ip range already exist")

type IPRepo struct {
	pool *pgxpool.Pool
}

func NewIPRepo(pool *pgxpool.Pool) *IPRepo {
	return &IPRepo{pool: pool}
}

func (r *IPRepo) AddIP(ctx context.Context, version int16, ipRange *net.IPNet, isWhitelist bool) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO ip (version, ip_range, is_whitelist) VALUES ($1, $2, $3)", version, ipRange, isWhitelist)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrIPExist
		}
		slog.Error("IPRepo AddIP: " + err.Error())
		return fmt.Errorf("IPRepo AddIP: %w", err)
	}
	slog.Debug("IPRepo AddIP success")
	return nil
}

func (r *IPRepo) DelIP(ctx context.Context, version int16, ipRange *net.IPNet) error {
	res, err := r.pool.Exec(ctx, "DELETE FROM ip WHERE version = $1 AND ip_range = $2", version, ipRange)
	if err != nil {
		slog.Error("IPRepo DelIP: " + err.Error())
		return err
	}
	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrIPNotFound
	}
	slog.Debug("IPRepo DelIP success")
	return nil
}

func (r *IPRepo) QueryIP(ctx context.Context, version int16, clientIP net.IP) (bool, error) {
	var isWhitelist bool

	sql := `
		SELECT is_whitelist
		FROM ip
		WHERE version = $1
			AND $2::inet <<= ip_range
	`

	row := r.pool.QueryRow(ctx, sql, version, clientIP.String())

	if err := row.Scan(&isWhitelist); err != nil {
		if err == pgx.ErrNoRows {
			return false, ErrIPNotFound
		}
		return false, fmt.Errorf("Error when scanning: %w", err)
	}
	return isWhitelist, nil
}

func (r *IPRepo) GetIPs(ctx context.Context) ([]model.IP, error) {
	var ips []model.IP

	rows, err := r.pool.Query(ctx, "SELECT * FROM ip")

	if err != nil {
		slog.Error("GetIPs error: " + err.Error())
		return nil, fmt.Errorf("GetIPs error %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ip model.IP
		if err := rows.Scan(&ip.ID, &ip.Version, &ip.IPRange, &ip.IsWhitelist, &ip.CreateAt, &ip.UpdateAt, &ip.Note); err != nil {
			slog.Error("GetIPs error: " + err.Error())
			return nil, fmt.Errorf("GetIPs error %w", err)
		}
		ips = append(ips, ip)
	}
	if err := rows.Err(); err != nil {
		slog.Error("GetIPs error: " + err.Error())
		return nil, fmt.Errorf("GetIPs error %w", err)
	}
	return ips, nil
}
