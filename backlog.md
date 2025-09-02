#MVP

Aquí tienes un **backlog de tareas exactas** para el MVP de tu app de finanzas personales, priorizado para asegurar que el producto mínimo viable sea lanzable, funcional y listo para recibir feedback real de usuarios:

***

## Backlog de Tareas MVP – Fintech App


***

### Configuración Inicial y Entorno

- [X] Crear repositorio y estructura base Golang + Flutter
- [X] Configurar entorno de desarrollo y base de datos PostgreSQL
- [X] Configurar CI/CD con pruebas básicas y despliegue local


### Gestión de Usuarios

- [X] Endpoints backend: registro, login, cierre de sesión, validación de sesión (JWT)
- [X] Frontend: pantallas de registro, login y navegación básica


### Configuración de Presupuesto Mensual

- [X] Backend: modelos y endpoints para presupuesto total y por categorías (crear, leer, actualizar)
- [X] Frontend: pantalla donde el usuario define su presupuesto total y lo asigna por categorías


### Cálculo y Gestión de Límites Diarios

- [X] Lógica backend para cálculo automático de límite diario por categoría
- [X] Registro y cálculo de "rollover" diario de saldo no gastado
- [X] Endpoint para consultar límite diario y saldo acumulado actual por categoría


### Registro de Gastos

- [X] Backend: endpoint para registrar gasto manual (categoría, monto, fecha)
- [X] Frontend: formulario simple para capturar gasto manual y asignarlo a categoría
- [X] Backend y frontend: historial básico de gastos (listado, filtro por fecha/categoría)


### Visualización y Alertas

- [X] Backend: endpoint para resumen de gastos, límites y saldo por categoría/día/mes
- [ ] Frontend: dashboard con barras o gráficos simples de gasto vs. presupuesto por categoría
- [ ] Frontend: mostrar alertas/avisos visuales si se superan los límites diarios o mensuales


### Seguridad y Gestión de Datos

- [ ] Hashing seguro de contraseñas (bcrypt)
- [ ] Manejo seguro de sesiones (JWT, refresh tokens)
- [ ] Validación de roles y acceso seguro a datos de cada usuario


### Pruebas y Validación

- [ ] Pruebas unitarias para modelos, lógica de límites y endpoints clave (backend)
- [ ] Pruebas funcionales en frontend (navegación, flujos principales)
- [ ] Pruebas de integración “happy path” (registrar usuario, presupuesto, gasto, visualizar dashboard)


### Extras recomendados (para cerrar el ciclo MVP)

- [ ] Documentar endpoints principales en Swagger/OpenAPI
- [ ] Instructivo muy simple para instalar y probar el MVP localmente


