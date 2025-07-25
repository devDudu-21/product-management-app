// Package repositories contains data access layer implementations for the product management application.
package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"product-management-app/core/dto"
	"product-management-app/core/models"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProductRepository handles database operations for products.
type ProductRepository struct {
	db  *sql.DB
	ctx context.Context
}

// NewProductRepository creates a new ProductRepository instance.
func NewProductRepository(ctx context.Context, db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db, ctx: ctx}
}

// Create creates a new product in the database.
func (r *ProductRepository) Create(createProductDTO dto.CreateProductDTO) (*models.Product, error) {
	res, err := r.db.Exec("INSERT INTO products(name, price, category, stock, description, image_url) VALUES(?, ?, ?, ?, ?, ?)", createProductDTO.Name, createProductDTO.Price, createProductDTO.Category, createProductDTO.Stock, createProductDTO.Description, createProductDTO.ImageURL)
	if err != nil {
		runtime.LogError(r.ctx, fmt.Sprintf("Failed to create product: %v", err))
		return nil, fmt.Errorf("failed to create product: %w", err)
	}
	id, _ := res.LastInsertId()

	var category, description, imageURL *string
	if createProductDTO.Category != "" {
		category = &createProductDTO.Category
	}
	if createProductDTO.Description != "" {
		description = &createProductDTO.Description
	}
	if createProductDTO.ImageURL != "" {
		imageURL = &createProductDTO.ImageURL
	}

	product := &models.Product{
		ID:          int(id),
		Name:        createProductDTO.Name,
		Price:       createProductDTO.Price,
		Category:    category,
		Stock:       createProductDTO.Stock,
		Description: description,
		ImageURL:    imageURL,
	}
	runtime.LogInfo(r.ctx, fmt.Sprintf("Product created: %+v", product))
	return product, nil
}

// GetByID retrieves a product by its ID.
func (r *ProductRepository) GetByID(id int) (*models.Product, error) {
	var category, description, imageURL, updatedAt sql.NullString
	var createdAt string

	row := r.db.QueryRow("SELECT id, name, price, category, stock, description, image_url, created_at, updated_at FROM products WHERE id = ?", id)

	product := &models.Product{}
	err := row.Scan(
		&product.ID,
		&product.Name,
		&product.Price,
		&category,
		&product.Stock,
		&description,
		&imageURL,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to fetch product: %w", err)
	}

	// Convertemos os NullString para ponteiros de string
	if category.Valid {
		product.Category = &category.String
	}
	if description.Valid {
		product.Description = &description.String
	}
	if imageURL.Valid {
		product.ImageURL = &imageURL.String
	}

	product.CreatedAt = createdAt

	if updatedAt.Valid {
		product.UpdatedAt = &updatedAt.String
	}

	return product, nil
}

// GetAll retrieves all products with pagination.
func (r *ProductRepository) GetAll(params dto.PaginationDTO) (*dto.PaginationResponse, error) {
	offset := (params.Page - 1) * params.PageSize
	rows, err := r.db.Query("SELECT id, name, price, category, stock, description, image_url, created_at, updated_at FROM products LIMIT ? OFFSET ?", params.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			runtime.LogError(r.ctx, fmt.Sprintf("Failed to close rows: %v", err))
		}
	}()

	products := []*models.Product{}
	for rows.Next() {
		var category, description, imageURL, updatedAt sql.NullString
		var createdAt string
		var id, stock int
		var name string
		var price float64

		if err := rows.Scan(&id, &name, &price, &category, &stock, &description, &imageURL, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		product := &models.Product{
			ID:        id,
			Name:      name,
			Price:     price,
			Stock:     stock,
			CreatedAt: createdAt,
		}

		if category.Valid {
			product.Category = &category.String
		}
		if description.Valid {
			product.Description = &description.String
		}
		if imageURL.Valid {
			product.ImageURL = &imageURL.String
		}
		if updatedAt.Valid {
			product.UpdatedAt = &updatedAt.String
		}

		products = append(products, product)
	}

	totalCount := 0
	err = r.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &dto.PaginationResponse{
		Products:   products,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}

// Update updates an existing product.
func (r *ProductRepository) Update(id int, name string, price float64) (*models.Product, error) {
	currentProduct, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	res, err := r.db.Exec("UPDATE products SET name = ?, price = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", name, price, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("product with ID %d not found", id)
	}

	currentProduct.Name = name
	currentProduct.Price = price

	var updatedAt string
	err = r.db.QueryRow("SELECT updated_at FROM products WHERE id = ?", id).Scan(&updatedAt)
	if err == nil {
		currentProduct.UpdatedAt = &updatedAt
	}

	return currentProduct, nil
}

// Delete removes a product from the database.
func (r *ProductRepository) Delete(id int) error {
	res, err := r.db.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("product with ID %d not found", id)
	}
	return nil
}
