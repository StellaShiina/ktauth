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

func (r *IPRepo) AddIP(ctx context.Context, version int16, ipRange *net.IPNet, isWhitelist bool, note *string) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO ip (version, ip_range, is_whitelist, note) VALUES ($1, $2, $3, $4)", version, ipRange, isWhitelist, note)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrIPExist
		}
		slog.Error("IPRepo AddIP: " + err.Error())
		return fmt.Errorf("IPRepo AddIP: %w", err)
	}
	slog.Debug("IPRepo AddIP success", "IPRange", ipRange.String())
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
	slog.Debug("IPRepo DelIP success", "IPRange", ipRange.String())
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

// Query version *int16, isWhiteList *bool if needed
func (r *IPRepo) GetIPs(ctx context.Context, version *int16, isWhiteList *bool) ([]model.IP, error) {
	var ips []model.IP
	var rows pgx.Rows
	var err error

	if version != nil && isWhiteList != nil {
		rows, err = r.pool.Query(ctx, "SELECT * FROM ip WHERE version = $1 AND is_whitelist = $2", *version, *isWhiteList)
	} else if version != nil {
		rows, err = r.pool.Query(ctx, "SELECT * FROM ip WHERE version = $1", *version)
	} else if isWhiteList != nil {
		rows, err = r.pool.Query(ctx, "SELECT * FROM ip WHERE is_whitelist = $1", *isWhiteList)
	} else {
		rows, err = r.pool.Query(ctx, "SELECT * FROM ip")
	}

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

func (r *IPRepo) UpdateIP(ctx context.Context, id int, isWhiteList bool, note *string) (model.IP, error) {
	var ip model.IP
	sql := `
		UPDATE ip
		SET is_whitelist = $1, note = $2
		WHERE id = $3
		RETURNING *
	`

	err := r.pool.QueryRow(ctx, sql, isWhiteList, note, id).Scan(&ip.ID, &ip.Version, &ip.IPRange, &ip.IsWhitelist, &ip.CreateAt, &ip.UpdateAt, &ip.Note)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ip, ErrIPNotFound
		}
		slog.Error("UpdateIP eeror: " + err.Error())
		return ip, fmt.Errorf("Error when updating: %w", err)
	}

	return ip, nil
}
