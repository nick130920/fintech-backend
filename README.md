# API Fintech - Backend

Backend para aplicación de finanzas personales construido con Go siguiendo **Clean Architecture**.

## Estructura del Proyecto

```
backend/
├── cmd/
│   └── server/          # Punto de entrada de la aplicación
│       └── main.go
├── internal/            # Lógica de negocio (no exportable)
│   ├── app/            # Configuración e inicialización de la app
│   ├── entity/         # Entidades de dominio
│   ├── usecase/        # Casos de uso (lógica de negocio)
│   │   └── repo/       # Interfaces de repositorios
│   └── controller/     # Controladores por protocolo
│       └── http/v1/    # Controladores HTTP v1
│           └── dto/    # Data Transfer Objects
├── pkg/                # Librerías reutilizables
│   ├── auth/          # Autenticación JWT
│   ├── database/      # Configuración de DB
│   ├── repository/    # Implementaciones de repositorios
│   └── validator/     # Validaciones
├── configs/           # Configuraciones
└── api/              # Especificaciones API
    └── swagger/
```

## Características

- **Arquitectura**: Clean Architecture siguiendo principios de [evrone/go-clean-template](https://github.com/evrone/go-clean-template)
- **Framework**: Gin-Gonic para APIs REST rápidas
- **Base de datos**: PostgreSQL con GORM
- **Autenticación**: JWT tokens con refresh token
- **Documentación**: Swagger/OpenAPI automática
- **Validación**: Validadores personalizados en español
- **Escalabilidad**: Inyección de dependencias y separación de responsabilidades

## Clean Architecture

### Capas de la Aplicación

#### 📋 Entidades (`internal/entity/`)
Contienen la lógica de negocio central y las reglas empresariales. Son independientes de frameworks externos.

#### 🎯 Casos de Uso (`internal/usecase/`)
Contienen la lógica de aplicación específica. Coordinan el flujo de datos hacia y desde las entidades.

#### 🌐 Controladores (`internal/controller/`)
Manejan la interfaz externa (HTTP, gRPC, etc.). Convierten los datos externos al formato que requieren los casos de uso.

#### 💾 Infraestructura (`pkg/`)
Implementaciones concretas de interfaces definidas en las capas internas (base de datos, servicios externos, etc.).

### Principios Aplicados

- **Inversión de Dependencias**: Las capas internas definen interfaces que implementan las capas externas
- **Separación de Responsabilidades**: Cada capa tiene una responsabilidad específica
- **Independencia de Frameworks**: La lógica de negocio no depende de frameworks externos
- **Testabilidad**: Fácil testing mediante inyección de dependencias e interfaces

## Dominios

### Usuario (User)
- Registro y autenticación
- Gestión de perfil
- Control de sesiones

### Cuenta (Account)
- Múltiples tipos: corriente, ahorros, crédito, inversión, efectivo
- Gestión de balances
- Configuración de alertas

### Transacción (Transaction)
- Ingresos, gastos y transferencias
- Categorización
- Filtros y búsquedas avanzadas

## Configuración

### Variables de Entorno

```bash
# Servidor
PORT=8080
GIN_MODE=debug
HOST=localhost

# Base de datos
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=fintech_db
DB_SSLMODE=disable
DB_AUTO_MIGRATE=true

# JWT
JWT_SECRET_KEY=tu-clave-secreta-muy-segura
JWT_EXPIRES_IN=3600

# Gmail / correo como entrada (OAuth2, solo lectura)
# Redirect URI debe coincidir con Google Cloud Console, p. ej.:
# https://tu-api/api/v1/email-connections/gmail/callback
GMAIL_CLIENT_ID=
GMAIL_CLIENT_SECRET=
GMAIL_REDIRECT_URL=
# Obligatorias si las tres de arriba están definidas (Railway/producción con Gmail). No sustituir por JWT.
TOKEN_ENCRYPTION_KEY=   # cadena fuerte; se hashea a 32 bytes para AES-GCM del refresh_token
OAUTH_STATE_SECRET=     # secreto HMAC para el parámetro state del OAuth
# Opcional: query inicial de búsqueda Gmail (p. ej. newer_than:90d)
GMAIL_DEFAULT_LIST_QUERY=

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:4200
```

**Privacidad (correo Gmail):** el usuario autoriza lectura de Gmail para detectar movimientos bancarios; el texto relevante puede enviarse al mismo flujo de IA que los SMS. No subas credenciales OAuth (`client_secret*.json`) al repositorio.

## Instalación y Uso

### Prerrequisitos

- Go 1.21+
- PostgreSQL 12+

### Pasos

1. **Clonar el repositorio**
   ```bash
   git clone <repo-url>
   cd backend
   ```

2. **Instalar dependencias**
   ```bash
   go mod tidy
   ```

3. **Configurar variables de entorno**
   ```bash
   # Crear archivo .env basado en las variables mostradas arriba
   ```

4. **Configurar base de datos**
   ```bash
   # Crear base de datos PostgreSQL
   createdb fintech_db
   ```

5. **Ejecutar migraciones**
   ```bash
   # Las migraciones se ejecutan automáticamente si DB_AUTO_MIGRATE=true
   ```

6. **Ejecutar servidor**
   ```bash
   go run cmd/server/main.go
   ```

### Acceso

- **API**: http://localhost:8080
- **Documentación Swagger**: http://localhost:8080/swagger/index.html
- **Health Check**: http://localhost:8080/health

## API Endpoints

### Autenticación
- `POST /api/v1/auth/register` - Registro de usuario
- `POST /api/v1/auth/login` - Inicio de sesión
- `POST /api/v1/auth/refresh` - Renovar token

### Usuarios
- `GET /api/v1/users/profile` - Obtener perfil
- `PUT /api/v1/users/profile` - Actualizar perfil

### Cuentas
- `GET /api/v1/accounts` - Listar cuentas
- `POST /api/v1/accounts` - Crear cuenta
- `GET /api/v1/accounts/:id` - Obtener cuenta
- `PUT /api/v1/accounts/:id` - Actualizar cuenta
- `DELETE /api/v1/accounts/:id` - Eliminar cuenta

### Transacciones
- `GET /api/v1/transactions` - Listar transacciones
- `POST /api/v1/transactions` - Crear transacción
- `GET /api/v1/transactions/:id` - Obtener transacción
- `PUT /api/v1/transactions/:id` - Actualizar transacción
- `DELETE /api/v1/transactions/:id` - Eliminar transacción

### Viajes (Trips)

Módulo para planificar viajes con presupuesto estimado, gastos compartidos
estilo Splitwise (con miembros reales o "fantasmas"), simplificación de
deudas, settlements, itinerario y reportes exportables.

- `GET /api/v1/trips` — Listar viajes (filtra por `?status=`)
- `POST /api/v1/trips` — Crear viaje
- `GET /api/v1/trips/:id` — Obtener viaje
- `PUT /api/v1/trips/:id` — Actualizar viaje
- `DELETE /api/v1/trips/:id` — Eliminar viaje
- `POST /api/v1/trips/:id/start|complete|cancel` — Cambiar estado

#### Miembros e invitaciones
- `GET/POST /api/v1/trips/:id/members`
- `PUT/DELETE /api/v1/trips/:id/members/:memberId`
- `GET/POST /api/v1/trips/:id/invitations`
- `POST /api/v1/trips/invitations/accept` — Aceptar token (auth)

#### Presupuesto y gastos
- `GET/PUT /api/v1/trips/:id/budget`
- `GET/POST /api/v1/trips/:id/expenses`
- `PUT/DELETE /api/v1/trips/:id/expenses/:expenseId`

#### Balance y settlements
- `GET /api/v1/trips/:id/balance` — Saldo neto + transferencias sugeridas
- `GET/POST /api/v1/trips/:id/settlements`
- `DELETE /api/v1/trips/:id/settlements/:settlementId`

#### Itinerario
- `GET/POST /api/v1/trips/:id/itinerary`
- `PUT/DELETE /api/v1/trips/:id/itinerary/:itemId`
- `POST /api/v1/trips/:id/itinerary/:itemId/link-expense`

#### Reportes e importación
- `GET /api/v1/trips/:id/report` — Reporte estructurado
- `GET /api/v1/trips/:id/report/export?format=csv|pdf` — Descargar
- `GET/POST /api/v1/trips/:id/import-suggestions` — Asignar gastos del banco

La simplificación de deudas vive en `pkg/finance/debtsimplify/` y los
totales/balances usan `pkg/exchange` para conversión multi-moneda.

## Desarrollo

### Generar documentación Swagger

```bash
# Instalar swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generar docs
swag init -g cmd/server/main.go -o api/swagger
```

### Ejecutar tests

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Estructura de Datos

### Usuario de ejemplo
```json
{
  "id": 1,
  "first_name": "Juan",
  "last_name": "Pérez",
  "email": "juan@ejemplo.com",
  "is_active": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### Cuenta de ejemplo
```json
{
  "id": 1,
  "name": "Cuenta Principal",
  "type": "checking",
  "balance": 10000.00,
  "currency": "COP",
  "is_active": true
}
```

### Transacción de ejemplo
```json
{
  "id": 1,
  "type": "expense",
  "amount": 250.00,
  "description": "Compra supermercado",
  "transaction_date": "2024-01-15T10:30:00Z",
  "category_name": "Alimentación"
}
```

## Contribución

1. Fork el proyecto
2. Crea una rama para tu feature
3. Commit tus cambios
4. Push a la rama
5. Abre un Pull Request

## Licencia

MIT License
