package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PrivacyHandler handles privacy policy requests
type PrivacyHandler struct{}

// NewPrivacyHandler creates a new privacy handler
func NewPrivacyHandler() *PrivacyHandler {
	return &PrivacyHandler{}
}

// GetPrivacyPolicy returns the privacy policy in HTML format
func (h *PrivacyHandler) GetPrivacyPolicy(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Política de Privacidad - Money Flow</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            color: #333;
        }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .last-updated { background: #ecf0f1; padding: 10px; border-radius: 5px; margin-bottom: 20px; }
        .contact { background: #e8f5e8; padding: 15px; border-radius: 5px; margin-top: 20px; }
    </style>
</head>
<body>
    <h1>Política de Privacidad - Money Flow</h1>
    
    <div class="last-updated">
        <strong>Última actualización:</strong> 24 de octubre de 2025
    </div>

    <h2>1. Información que Recopilamos</h2>
    <p>Money Flow recopila la siguiente información para brindarte nuestros servicios:</p>
    <ul>
        <li><strong>Información de cuenta:</strong> Email, nombre de usuario y contraseña encriptada</li>
        <li><strong>Datos financieros:</strong> Transacciones, categorías, presupuestos y metas financieras</li>
        <li><strong>Información de dispositivo:</strong> Tipo de dispositivo, sistema operativo, identificadores únicos</li>
        <li><strong>Datos de ubicación:</strong> Solo para detección automática de moneda (opcional)</li>
        <li><strong>Notificaciones bancarias:</strong> SMS y notificaciones para procesamiento automático (opcional)</li>
    </ul>

    <h2>2. Cómo Utilizamos tu Información</h2>
    <p>Utilizamos tu información para:</p>
    <ul>
        <li>Proporcionar y mejorar nuestros servicios de gestión financiera</li>
        <li>Procesar y categorizar automáticamente tus transacciones</li>
        <li>Generar reportes y análisis personalizados</li>
        <li>Enviar notificaciones importantes sobre tu cuenta</li>
        <li>Detectar y prevenir actividades fraudulentas</li>
        <li>Cumplir con obligaciones legales y regulatorias</li>
    </ul>

    <h2>3. Compartir Información</h2>
    <p><strong>No vendemos, alquilamos ni compartimos tu información personal con terceros</strong> excepto en los siguientes casos:</p>
    <ul>
        <li>Con tu consentimiento explícito</li>
        <li>Para cumplir con la ley o procesos legales</li>
        <li>Para proteger nuestros derechos, propiedad o seguridad</li>
        <li>Con proveedores de servicios que nos ayudan a operar la aplicación (bajo estrictos acuerdos de confidencialidad)</li>
    </ul>

    <h2>4. Seguridad de los Datos</h2>
    <p>Implementamos medidas de seguridad robustas:</p>
    <ul>
        <li>Encriptación de datos en tránsito y en reposo</li>
        <li>Autenticación de dos factores disponible</li>
        <li>Acceso restringido a datos personales</li>
        <li>Monitoreo continuo de seguridad</li>
        <li>Auditorías regulares de seguridad</li>
    </ul>

    <h2>5. Retención de Datos</h2>
    <p>Conservamos tu información mientras:</p>
    <ul>
        <li>Tu cuenta esté activa</li>
        <li>Sea necesario para proporcionar nuestros servicios</li>
        <li>Sea requerido por ley</li>
    </ul>
    <p>Puedes solicitar la eliminación de tu cuenta y datos en cualquier momento.</p>

    <h2>6. Tus Derechos</h2>
    <p>Tienes derecho a:</p>
    <ul>
        <li>Acceder a tu información personal</li>
        <li>Corregir datos inexactos</li>
        <li>Solicitar la eliminación de tu cuenta</li>
        <li>Exportar tus datos</li>
        <li>Retirar consentimientos otorgados</li>
        <li>Presentar quejas ante autoridades de protección de datos</li>
    </ul>

    <h2>7. Menores de Edad</h2>
    <p>Money Flow no está dirigido a menores de 13 años. No recopilamos conscientemente información personal de menores de 13 años. Si descubrimos que hemos recopilado información de un menor, la eliminaremos inmediatamente.</p>

    <h2>8. Cambios en esta Política</h2>
    <p>Podemos actualizar esta política ocasionalmente. Te notificaremos sobre cambios significativos a través de la aplicación o por email.</p>

    <h2>9. Transferencias Internacionales</h2>
    <p>Tus datos pueden ser procesados en servidores ubicados fuera de tu país de residencia. Garantizamos que estas transferencias cumplan con las leyes de protección de datos aplicables.</p>

    <div class="contact">
        <h2>10. Contacto</h2>
        <p>Si tienes preguntas sobre esta política de privacidad, contáctanos:</p>
        <ul>
            <li><strong>Email:</strong> privacy@moneyflow.app</li>
            <li><strong>Dirección:</strong> [Tu dirección comercial]</li>
            <li><strong>Teléfono:</strong> [Tu número de contacto]</li>
        </ul>
    </div>
</body>
</html>
	`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// GetTermsOfService returns the terms of service in HTML format
func (h *PrivacyHandler) GetTermsOfService(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Términos de Servicio - Money Flow</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            color: #333;
        }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .last-updated { background: #ecf0f1; padding: 10px; border-radius: 5px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <h1>Términos de Servicio - Money Flow</h1>
    
    <div class="last-updated">
        <strong>Última actualización:</strong> 24 de octubre de 2025
    </div>

    <h2>1. Aceptación de Términos</h2>
    <p>Al usar Money Flow, aceptas estos términos de servicio. Si no estás de acuerdo, no uses la aplicación.</p>

    <h2>2. Descripción del Servicio</h2>
    <p>Money Flow es una aplicación de gestión financiera personal que te ayuda a:</p>
    <ul>
        <li>Rastrear ingresos y gastos</li>
        <li>Crear y gestionar presupuestos</li>
        <li>Analizar patrones de gasto</li>
        <li>Procesar automáticamente notificaciones bancarias</li>
    </ul>

    <h2>3. Responsabilidades del Usuario</h2>
    <p>Te comprometes a:</p>
    <ul>
        <li>Proporcionar información precisa y actualizada</li>
        <li>Mantener la seguridad de tu cuenta</li>
        <li>No usar la aplicación para actividades ilegales</li>
        <li>Respetar los derechos de propiedad intelectual</li>
    </ul>

    <h2>4. Limitación de Responsabilidad</h2>
    <p>Money Flow se proporciona "tal como está". No garantizamos que el servicio sea ininterrumpido o libre de errores.</p>

    <h2>5. Contacto</h2>
    <p>Para preguntas sobre estos términos: <strong>legal@moneyflow.app</strong></p>
</body>
</html>
	`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
