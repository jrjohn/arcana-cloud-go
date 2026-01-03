package graphql

import (
	"github.com/graphql-go/graphql"
)

// Schema represents the GraphQL schema
type Schema struct {
	schema graphql.Schema
}

// BuildSchema builds the GraphQL schema
func BuildSchema(resolver *Resolver) (*Schema, error) {
	// User type
	userType := graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"username": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"email": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"firstName": &graphql.Field{
				Type: graphql.String,
			},
			"lastName": &graphql.Field{
				Type: graphql.String,
			},
			"role": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"isActive": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
			},
			"isVerified": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
			},
			"createdAt": &graphql.Field{
				Type: graphql.NewNonNull(graphql.DateTime),
			},
			"updatedAt": &graphql.Field{
				Type: graphql.NewNonNull(graphql.DateTime),
			},
		},
	})

	// Auth response type
	authResponseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "AuthResponse",
		Fields: graphql.Fields{
			"accessToken": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"refreshToken": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"tokenType": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"expiresIn": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"user": &graphql.Field{
				Type: graphql.NewNonNull(userType),
			},
		},
	})

	// Page info type
	pageInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PageInfo",
		Fields: graphql.Fields{
			"page": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"size": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"totalItems": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
			"totalPages": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
			},
		},
	})

	// Users connection type
	usersConnectionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "UsersConnection",
		Fields: graphql.Fields{
			"items": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(userType))),
			},
			"pageInfo": &graphql.Field{
				Type: graphql.NewNonNull(pageInfoType),
			},
		},
	})

	// Plugin type
	pluginType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Plugin",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"key": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"name": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"version": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"state": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"enabled": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
			},
		},
	})

	// Query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			// User queries
			"me": &graphql.Field{
				Type:        userType,
				Description: "Get current authenticated user",
				Resolve:     resolver.Me,
			},
			"user": &graphql.Field{
				Type:        userType,
				Description: "Get user by ID",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: resolver.User,
			},
			"users": &graphql.Field{
				Type:        usersConnectionType,
				Description: "List users with pagination",
				Args: graphql.FieldConfigArgument{
					"page": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 1,
					},
					"size": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 10,
					},
				},
				Resolve: resolver.Users,
			},

			// Plugin queries
			"plugins": &graphql.Field{
				Type:        graphql.NewList(pluginType),
				Description: "List all plugins",
				Resolve:     resolver.Plugins,
			},
			"plugin": &graphql.Field{
				Type:        pluginType,
				Description: "Get plugin by key",
				Args: graphql.FieldConfigArgument{
					"key": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: resolver.Plugin,
			},
		},
	})

	// Input types
	registerInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "RegisterInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"username": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"email": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"password": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"firstName": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"lastName": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
		},
	})

	loginInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "LoginInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"usernameOrEmail": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
			"password": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.String),
			},
		},
	})

	updateProfileInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "UpdateProfileInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"firstName": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"lastName": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"email": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
		},
	})

	// Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			// Auth mutations
			"register": &graphql.Field{
				Type:        authResponseType,
				Description: "Register a new user",
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(registerInputType),
					},
				},
				Resolve: resolver.Register,
			},
			"login": &graphql.Field{
				Type:        authResponseType,
				Description: "Login user",
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(loginInputType),
					},
				},
				Resolve: resolver.Login,
			},
			"refreshToken": &graphql.Field{
				Type:        authResponseType,
				Description: "Refresh access token",
				Args: graphql.FieldConfigArgument{
					"refreshToken": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: resolver.RefreshToken,
			},
			"logout": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Logout current session",
				Resolve:     resolver.Logout,
			},

			// User mutations
			"updateProfile": &graphql.Field{
				Type:        userType,
				Description: "Update current user profile",
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(updateProfileInputType),
					},
				},
				Resolve: resolver.UpdateProfile,
			},
			"changePassword": &graphql.Field{
				Type:        graphql.Boolean,
				Description: "Change password",
				Args: graphql.FieldConfigArgument{
					"currentPassword": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"newPassword": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: resolver.ChangePassword,
			},

			// Plugin mutations
			"enablePlugin": &graphql.Field{
				Type:        pluginType,
				Description: "Enable a plugin",
				Args: graphql.FieldConfigArgument{
					"key": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: resolver.EnablePlugin,
			},
			"disablePlugin": &graphql.Field{
				Type:        pluginType,
				Description: "Disable a plugin",
				Args: graphql.FieldConfigArgument{
					"key": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: resolver.DisablePlugin,
			},
		},
	})

	// Build schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
	if err != nil {
		return nil, err
	}

	return &Schema{schema: schema}, nil
}

// Schema returns the graphql.Schema
func (s *Schema) Schema() graphql.Schema {
	return s.schema
}
