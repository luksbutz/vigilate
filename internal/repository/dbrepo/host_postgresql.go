package dbrepo

import (
	"context"
	"github.com/luksbutz/vigilate/internal/models"
	"time"
)

// InsertHost inserts a host into the database
func (m *postgresDBRepo) InsertHost(h models.Host) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `insert into hosts (
                   host_name, canonical_name, url, ip, ipv6, location,
                   os, active, created_at, updated_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) returning id`

	var newID int
	err := m.DB.QueryRowContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		time.Now()).Scan(&newID)
	if err != nil {
		return 0, err
	}

	return newID, nil
}

// GetHostByID returns a host by id
func (m *postgresDBRepo) GetHostByID(id int) (models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at
			from hosts where id = $1`

	var host models.Host
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&host.ID,
		&host.HostName,
		&host.CanonicalName,
		&host.URL,
		&host.IP,
		&host.IPV6,
		&host.Location,
		&host.OS,
		&host.Active,
		&host.CreatedAt,
		&host.UpdatedAt,
	)

	return host, err
}

// UpdateHost updates a host
func (m *postgresDBRepo) UpdateHost(h models.Host) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
update hosts set host_name = $1, canonical_name = $2, url = $3, ip = $4,
                 ipv6 = $5, location = $6, os = $7, active = $8, updated_at = $9 where id = $10`

	_, err := m.DB.ExecContext(ctx, stmt,
		h.HostName,
		h.CanonicalName,
		h.URL,
		h.IP,
		h.IPV6,
		h.Location,
		h.OS,
		h.Active,
		time.Now(),
		h.ID,
	)

	return err
}

// AllHosts returns a slice with all hosts
func (m *postgresDBRepo) AllHosts() ([]models.Host, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
select id, host_name, canonical_name, url, ip, ipv6, location, os, active, created_at, updated_at from hosts
`

	var hosts []models.Host

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h models.Host
		err = rows.Scan(
			&h.ID,
			&h.HostName,
			&h.CanonicalName,
			&h.URL,
			&h.IP,
			&h.IPV6,
			&h.Location,
			&h.OS,
			&h.Active,
			&h.CreatedAt,
			&h.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		hosts = append(hosts, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}
