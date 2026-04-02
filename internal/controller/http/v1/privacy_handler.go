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

// GetPrivacyPolicy godoc
// @Summary Política de privacidad
// @Description Devuelve la política de privacidad en formato HTML
// @Tags legal
// @Produce html
// @Success 200 {string} string
// @Router /privacy [get]
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

// GetTermsOfService godoc
// @Summary Términos de servicio
// @Description Devuelve los términos de servicio en formato HTML
// @Tags legal
// @Produce html
// @Success 200 {string} string
// @Router /terms [get]
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

// GetAccountDeletion godoc
// @Summary Eliminación de cuenta
// @Description Devuelve la página informativa para eliminación de cuenta en formato HTML
// @Tags legal
// @Produce html
// @Success 200 {string} string
// @Router /account-deletion [get]
func (h *PrivacyHandler) GetAccountDeletion(c *gin.Context) {
	html := `
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Eliminación de Cuenta - Money Flow</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            color: #333;
            background-color: #f8f9fa;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 { 
            color: #e74c3c; 
            border-bottom: 3px solid #e74c3c; 
            padding-bottom: 10px;
            text-align: center;
        }
        h2 { color: #34495e; margin-top: 30px; }
        .warning {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .steps {
            background: #e8f5e8;
            padding: 20px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .contact-form {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 5px;
            margin: 20px 0;
            border: 2px solid #3498db;
        }
        .data-table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .data-table th, .data-table td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        .data-table th {
            background-color: #f2f2f2;
            font-weight: bold;
        }
        .retention-immediate { color: #e74c3c; font-weight: bold; }
        .retention-delayed { color: #f39c12; font-weight: bold; }
        .retention-legal { color: #9b59b6; font-weight: bold; }
        .button {
            display: inline-block;
            background: #e74c3c;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 5px;
            font-weight: bold;
            margin: 10px 0;
        }
        .button:hover {
            background: #c0392b;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🗑️ Eliminación de Cuenta - Money Flow</h1>
        
        <div class="warning">
            <strong>⚠️ ADVERTENCIA IMPORTANTE:</strong><br>
            La eliminación de tu cuenta de Money Flow es <strong>permanente e irreversible</strong>. 
            Una vez procesada, no podremos recuperar tus datos financieros, transacciones, 
            presupuestos o cualquier otra información asociada a tu cuenta.
        </div>

        <h2>📋 Pasos para Solicitar la Eliminación de tu Cuenta</h2>
        
        <div class="steps">
            <h3>Sigue estos pasos para eliminar tu cuenta de Money Flow:</h3>
            <ol>
                <li><strong>Exporta tus datos</strong> (opcional pero recomendado):
                    <ul>
                        <li>Abre la app Money Flow</li>
                        <li>Ve a Configuración → Exportar Datos</li>
                        <li>Descarga una copia de tus transacciones y reportes</li>
                    </ul>
                </li>
                <li><strong>Envía tu solicitud de eliminación</strong> usando uno de estos métodos:
                    <ul>
                        <li>📧 <strong>Email:</strong> <a href="mailto:delete-account@moneyflow.app?subject=Solicitud de Eliminación de Cuenta&body=Hola, solicito la eliminación completa de mi cuenta de Money Flow.%0A%0AEmail de la cuenta: [TU_EMAIL]%0AFecha de solicitud: [FECHA]%0A%0AConfirmo que entiendo que esta acción es irreversible.%0A%0AGracias.">delete-account@moneyflow.app</a></li>
                        <li>📱 <strong>Desde la app:</strong> Configuración → Cuenta → Eliminar Cuenta</li>
                        <li>🌐 <strong>Formulario web:</strong> <a href="#contact-form">Usar formulario abajo</a></li>
                    </ul>
                </li>
                <li><strong>Verificación de identidad:</strong>
                    <ul>
                        <li>Te enviaremos un email de confirmación</li>
                        <li>Debes confirmar la eliminación dentro de 7 días</li>
                        <li>Si no confirmas, la solicitud será cancelada</li>
                    </ul>
                </li>
                <li><strong>Procesamiento:</strong>
                    <ul>
                        <li>Una vez confirmada, procesaremos tu solicitud en 30 días</li>
                        <li>Recibirás una confirmación final cuando esté completa</li>
                    </ul>
                </li>
            </ol>
        </div>

        <h2>🗂️ Datos que se Eliminan y Períodos de Retención</h2>
        
        <table class="data-table">
            <thead>
                <tr>
                    <th>Tipo de Datos</th>
                    <th>Descripción</th>
                    <th>Tiempo de Eliminación</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td><strong>Información Personal</strong></td>
                    <td>Nombre, email, preferencias de usuario</td>
                    <td class="retention-immediate">Inmediato (24-48 horas)</td>
                </tr>
                <tr>
                    <td><strong>Datos Financieros</strong></td>
                    <td>Transacciones, categorías, presupuestos, metas</td>
                    <td class="retention-immediate">Inmediato (24-48 horas)</td>
                </tr>
                <tr>
                    <td><strong>Configuraciones de App</strong></td>
                    <td>Preferencias, notificaciones, temas</td>
                    <td class="retention-immediate">Inmediato (24-48 horas)</td>
                </tr>
                <tr>
                    <td><strong>Datos de Cuentas Bancarias</strong></td>
                    <td>Información de cuentas vinculadas, patrones</td>
                    <td class="retention-immediate">Inmediato (24-48 horas)</td>
                </tr>
                <tr>
                    <td><strong>Logs de Actividad</strong></td>
                    <td>Registros de uso, accesos, errores</td>
                    <td class="retention-delayed">30 días (por seguridad)</td>
                </tr>
                <tr>
                    <td><strong>Datos de Facturación</strong></td>
                    <td>Historial de pagos, suscripciones</td>
                    <td class="retention-legal">7 años (obligación legal)</td>
                </tr>
                <tr>
                    <td><strong>Comunicaciones Legales</strong></td>
                    <td>Correspondencia sobre disputas, términos</td>
                    <td class="retention-legal">7 años (obligación legal)</td>
                </tr>
            </tbody>
        </table>

        <div class="warning">
            <h3>📝 Datos que NO se Eliminan (Obligaciones Legales):</h3>
            <ul>
                <li><strong>Registros de facturación:</strong> Conservados por 7 años según regulaciones fiscales</li>
                <li><strong>Comunicaciones legales:</strong> Conservadas según requerimientos legales</li>
                <li><strong>Datos agregados y anonimizados:</strong> Usados para estadísticas generales (sin identificación personal)</li>
            </ul>
        </div>

        <h2 id="contact-form">📧 Formulario de Solicitud de Eliminación</h2>
        
        <div class="contact-form">
            <p><strong>Para solicitar la eliminación de tu cuenta, envía un email con la siguiente información:</strong></p>
            
            <a href="mailto:delete-account@moneyflow.app?subject=Solicitud de Eliminación de Cuenta - Money Flow&body=Hola equipo de Money Flow,%0A%0ASolicito la eliminación completa de mi cuenta y todos los datos asociados.%0A%0A📧 Email de la cuenta: [ESCRIBE_TU_EMAIL_AQUÍ]%0A📅 Fecha de solicitud: $(new Date().toLocaleDateString())%0A%0A✅ Confirmo que:%0A- He leído y entiendo la política de eliminación%0A- Entiendo que esta acción es irreversible%0A- He exportado mis datos si los necesito (opcional)%0A%0A💬 Motivo de eliminación (opcional): [ESCRIBE_AQUÍ_SI_DESEAS]%0A%0AGracias por su atención.%0A%0ASaludos." 
               class="button">
                📧 Enviar Solicitud de Eliminación
            </a>
        </div>

        <h2>⏱️ Cronograma del Proceso</h2>
        <ul>
            <li><strong>Día 0:</strong> Recibes tu solicitud de eliminación</li>
            <li><strong>Día 1-2:</strong> Te enviamos email de confirmación</li>
            <li><strong>Día 7:</strong> Fecha límite para confirmar (si no confirmas, se cancela)</li>
            <li><strong>Día 8-30:</strong> Procesamiento de eliminación</li>
            <li><strong>Día 30:</strong> Confirmación final de eliminación completa</li>
        </ul>

        <div class="contact-form">
            <h3>💬 ¿Necesitas Ayuda?</h3>
            <p>Si tienes preguntas sobre el proceso de eliminación:</p>
            <ul>
                <li>📧 <strong>Email:</strong> support@moneyflow.app</li>
                <li>📧 <strong>Eliminación:</strong> delete-account@moneyflow.app</li>
            </ul>
        </div>
    </div>
</body>
</html>
	`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
