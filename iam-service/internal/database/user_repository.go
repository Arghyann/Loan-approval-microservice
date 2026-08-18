package database
import (                                                                                                                               
        "database/sql"                                                                                                                        
        "iam-service/internal/models"                                                                                                         
    )

 // UserRepository holds the connection to the database                                                                                 
    type UserRepository struct {                                                                                                           
        db *sql.DB                                                                                                                            
    }                                                                                                                                      
                                                                                                                                           
    // NewUserRepository is a constructor function that creates our repository                                                             
    func NewUserRepository(db *sql.DB) *UserRepository {                                                                                   
        return &UserRepository{                                                                                                               
            db: db,                                                                                                                               
        }
    }
  
    // Create takes the User model and runs the SQL INSERT command
    func (r *UserRepository) Create(user models.User) error {
        // 1. Write the SQL query. 
        // We use $1, $2 to prevent SQL Injection attacks!
        query := `
            INSERT INTO users (id, email, password_hash, role, created_at)
            VALUES ($1, $2, $3, $4, $5)
        `
        
        // 2. Execute the query using the data from the user model
        _, err := r.db.Exec(
            query, 
            user.ID, 
            user.Email, 
            user.PasswordHash, 
            user.Role, 
            user.CreatedAt,
        )
        
        // 3. Return any errors back to the handler
        return err
    }
