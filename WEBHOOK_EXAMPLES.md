# 🔗 Ejemplos de Webhooks - Sistema de Notificaciones Bancarias

## 🎯 Objetivo
Procesar automáticamente notificaciones SMS/Push de bancos y crear transacciones en la aplicación.

## 📋 Endpoints Disponibles

### 1. **Webhook Principal** - `/api/v1/webhooks/bank-notification`
Recibe notificaciones bancarias de cualquier canal.

```bash
POST /api/v1/webhooks/bank-notification
Content-Type: application/json

{
  "message": "Compra por $50,000 en SUPERMERCADO XYZ el 15/01/2024",
  "phone": "+573001234567",
  "channel": "sms",
  "received_at": "2024-01-15T10:30:00Z",
  "bank_code": "BANCOLOMBIA",
  "user_id": 1
}
```

### 2. **Webhook SMS** - `/api/v1/webhooks/sms`
Específico para notificaciones SMS.

```bash
POST /api/v1/webhooks/sms
Content-Type: application/json

{
  "message": "Retiro por $100,000 en CAJERO AUTOMATICO el 15/01/2024",
  "from": "+57123456789",
  "to": "+573001234567",
  "received_at": "2024-01-15T14:20:00Z",
  "provider": "twilio"
}
```

### 3. **Estadísticas** - `/api/v1/webhooks/stats`
Obtiene estadísticas de procesamiento.

```bash
GET /api/v1/webhooks/stats?user_id=1&days=30
```

## 🏦 Ejemplos por Banco

### **Bancolombia**
```json
{
  "message": "Compra por $75,500 en TIENDA LA REBAJA el 15/01/2024 a las 10:30",
  "phone": "+573001234567",
  "channel": "sms",
  "received_at": "2024-01-15T10:30:00Z",
  "bank_code": "BANCOLOMBIA"
}
```

**Respuesta esperada:**
```json
{
  "success": true,
  "transaction_created": true,
  "transaction_id": 123,
  "amount": 75500.0,
  "description": "TIENDA LA REBAJA",
  "confidence": 0.95,
  "requires_validation": false,
  "pattern_used": "Bancolombia - Compra"
}
```

### **Davivienda**
```json
{
  "message": "Transaccion aprobada por $45,000 en RESTAURANTE ANDRES 15/01/2024",
  "phone": "+573001234567",
  "channel": "sms",
  "received_at": "2024-01-15T12:15:00Z",
  "bank_code": "DAVIVIENDA"
}
```

### **Nequi**
```json
{
  "message": "Pagaste $25,000 a UBER",
  "phone": "+573001234567",
  "channel": "push",
  "received_at": "2024-01-15T18:45:00Z",
  "bank_code": "NEQUI"
}
```

### **Banco de Bogotá**
```json
{
  "message": "Compra: $120,000 ALMACENES EXITO 15/01/2024",
  "phone": "+573001234567",
  "channel": "sms",
  "received_at": "2024-01-15T16:20:00Z",
  "bank_code": "BANCO_BOGOTA"
}
```

## 🔄 Flujo de Procesamiento

### **1. Recepción**
- Webhook recibe notificación
- Valida formato y campos requeridos
- Registra en logs con request ID

### **2. Identificación de Usuario**
- Si `user_id` está presente → usar directamente
- Si no → buscar por `phone` en cuentas bancarias
- Si no se encuentra → error 404

### **3. Búsqueda de Patrones**
- Obtener patrones activos del usuario
- Filtrar por canal (sms, push, email)
- Ordenar por prioridad

### **4. Procesamiento**
- Aplicar regex de cada patrón
- Calcular confianza basada en coincidencias
- Seleccionar patrón con mayor confianza

### **5. Creación de Transacción**
- Si confianza ≥ threshold Y auto_approve = true → crear automáticamente
- Si no → marcar para validación manual
- Actualizar estadísticas del patrón

## 📊 Respuestas del Sistema

### **Éxito - Transacción Creada**
```json
{
  "success": true,
  "transaction_created": true,
  "transaction_id": 456,
  "amount": 50000.0,
  "description": "SUPERMERCADO XYZ",
  "confidence": 0.92,
  "requires_validation": false,
  "pattern_used": "Bancolombia - Compra",
  "extracted_data": {
    "amount": 50000.0,
    "merchant": "SUPERMERCADO XYZ",
    "date": "15/01/2024"
  }
}
```

### **Éxito - Requiere Validación**
```json
{
  "success": true,
  "transaction_created": false,
  "confidence": 0.65,
  "requires_validation": true,
  "reason": "Requiere validación manual (confianza insuficiente)",
  "pattern_used": "Bancolombia - Compra",
  "extracted_data": {
    "amount": 30000.0,
    "description": "COMERCIO DESCONOCIDO"
  }
}
```

### **Error - Sin Patrones**
```json
{
  "success": false,
  "transaction_created": false,
  "confidence": 0.0,
  "reason": "No hay patrones activos para este canal"
}
```

### **Error - Usuario No Encontrado**
```json
{
  "error": "not_found",
  "message": "No se encontró cuenta bancaria asociada al teléfono",
  "request_id": "a1b2c3d4"
}
```

## 🧪 Testing con cURL

### **Probar Notificación Bancolombia**
```bash
curl -X POST http://localhost:8080/api/v1/webhooks/bank-notification \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Compra por $50,000 en SUPERMERCADO XYZ el 15/01/2024",
    "phone": "+573001234567",
    "channel": "sms",
    "received_at": "2024-01-15T10:30:00Z",
    "user_id": 1
  }'
```

### **Probar SMS Davivienda**
```bash
curl -X POST http://localhost:8080/api/v1/webhooks/sms \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Transaccion aprobada por $75,500 en TIENDA ABC 15/01/2024",
    "from": "+57123456789",
    "to": "+573001234567",
    "received_at": "2024-01-15T14:20:00Z"
  }'
```

### **Obtener Estadísticas**
```bash
curl -X GET "http://localhost:8080/api/v1/webhooks/stats?user_id=1&days=7"
```

## 🔧 Configuración de Patrones

### **Crear Patrón Personalizado**
```bash
curl -X POST http://localhost:8080/api/v1/notification-patterns \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "bank_account_id": 1,
    "name": "Mi Banco - Compras",
    "description": "Patrón personalizado para mi banco",
    "channel": "sms",
    "message_pattern": "Pago de \\$([0-9,]+) en (.+)",
    "example_message": "Pago de $25,000 en TIENDA LOCAL",
    "amount_regex": "\\$([0-9,]+)",
    "merchant_regex": "en (.+)$",
    "confidence_threshold": 0.8,
    "auto_approve": true
  }'
```

## 🚀 Integración con Servicios

### **Twilio SMS**
```javascript
// Webhook endpoint para Twilio
app.post('/twilio-webhook', (req, res) => {
  const { Body, From, To } = req.body;
  
  // Reenviar a nuestro sistema
  fetch('http://localhost:8080/api/v1/webhooks/sms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: Body,
      from: From,
      to: To,
      received_at: new Date().toISOString(),
      provider: 'twilio'
    })
  });
  
  res.status(200).send('OK');
});
```

### **Firebase Push Notifications**
```javascript
// Procesar notificación push
function processPushNotification(notification) {
  fetch('http://localhost:8080/api/v1/webhooks/bank-notification', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: notification.body,
      channel: 'push',
      received_at: new Date().toISOString(),
      user_id: notification.userId
    })
  });
}
```

## 📱 Próximos Pasos

1. **Configurar Cuentas Bancarias** con teléfonos de notificación
2. **Crear Patrones** específicos para tus bancos
3. **Configurar Webhooks** con tu proveedor SMS/Push
4. **Probar** con notificaciones reales
5. **Revisar** transacciones creadas automáticamente
6. **Ajustar** patrones según sea necesario

---

**¡Tu sistema de notificaciones bancarias está listo! 🎉**

Ahora las transacciones se crearán automáticamente cuando recibas notificaciones de tus bancos.
