window.CommentWidget = (function() {
    let config = {
        apiUrl: 'https://comments.kuepper.nrw/api/comments',
        theme: 'light'
    };

    // CSS für das Widget (gleich wie vorher)
    const widgetCSS = `
        .comment-widget {
            margin: 20px 0;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        .comment-widget * {
            box-sizing: border-box;
        }

        .comment-form {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 30px;
        }

        .comment-form h3 {
            margin: 0 0 20px 0;
            color: #495057;
            font-size: 1.2rem;
        }

        .comment-form-group {
            margin-bottom: 15px;
        }

        .comment-form label {
            display: block;
            margin-bottom: 5px;
            color: #495057;
            font-weight: 500;
        }

        .comment-form input,
        .comment-form textarea {
            width: 100%;
            padding: 10px 12px;
            border: 1px solid #ced4da;
            border-radius: 4px;
            font-size: 14px;
            transition: border-color 0.15s ease-in-out;
        }

        .comment-form input:focus,
        .comment-form textarea:focus {
            outline: none;
            border-color: #007bff;
            box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
        }

        .comment-form textarea {
            min-height: 100px;
            resize: vertical;
        }

        .comment-submit-btn {
            background: #007bff;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.15s ease-in-out;
        }

        .comment-submit-btn:hover {
            background: #0056b3;
        }

        .comment-submit-btn:disabled {
            background: #6c757d;
            cursor: not-allowed;
        }

        .comment-message {
            padding: 10px 15px;
            border-radius: 4px;
            margin-bottom: 15px;
            font-size: 14px;
        }

        .comment-message.success {
            background: #d4edda;
            border: 1px solid #c3e6cb;
            color: #155724;
        }

        .comment-message.error {
            background: #f8d7da;
            border: 1px solid #f5c6cb;
            color: #721c24;
        }

        .comments-list {
            margin-top: 30px;
        }

        .comments-list h3 {
            margin: 0 0 20px 0;
            color: #495057;
            font-size: 1.2rem;
        }

        .comment-item {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 15px;
            margin-bottom: 15px;
        }

        .comment-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
            font-size: 13px;
        }

        .comment-author {
            font-weight: 600;
            color: #007bff;
        }

        .comment-date {
            color: #6c757d;
        }

        .comment-text {
            color: #495057;
            line-height: 1.5;
        }

        .comments-loading {
            text-align: center;
            color: #6c757d;
            padding: 20px;
            font-style: italic;
        }

        .comments-empty {
            text-align: center;
            color: #6c757d;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
            border: 1px dashed #dee2e6;
        }

        @media (max-width: 768px) {
            .comment-form {
                padding: 15px;
            }
            
            .comment-header {
                flex-direction: column;
                align-items: flex-start;
            }
            
            .comment-date {
                margin-top: 5px;
            }
        }
    `;

    // CSS einmalig hinzufügen
    function injectCSS() {
        if (!document.getElementById('comment-widget-styles')) {
            const style = document.createElement('style');
            style.id = 'comment-widget-styles';
            style.textContent = widgetCSS;
            document.head.appendChild(style);
        }
    }

    // HTML für das Widget generieren
    function createWidgetHTML(postId) {
        return `
            <div class="comment-widget" data-post-id="${postId}">
                <div class="comment-form">
                    <h3>💬 Kommentar schreiben</h3>
                    <div class="comment-message-container"></div>
                    <form class="comment-form-element">
                        <div class="comment-form-group">
                            <label for="username-${postId}">Name *</label>
                            <input type="text" id="username-${postId}" name="username" required>
                        </div>
                        <div class="comment-form-group">
                            <label for="email-${postId}">E-Mail *</label>
                            <input type="email" id="email-${postId}" name="mailaddress" required>
                        </div>
                        <div class="comment-form-group">
                            <label for="text-${postId}">Kommentar *</label>
                            <textarea id="text-${postId}" name="text" required placeholder="Schreibe hier deinen Kommentar..."></textarea>
                        </div>
                        <button type="submit" class="comment-submit-btn">Kommentar absenden</button>
                    </form>
                </div>
                <div class="comments-list">
                    <h3>📝 Kommentare</h3>
                    <div class="comments-container">
                        <div class="comments-loading">Kommentare werden geladen...</div>
                    </div>
                </div>
            </div>
        `;
    }

    // Post-ID aus verschiedenen Quellen ableiten
    function getPostIdFromUrl() {
        const path = window.location.pathname;
        const segments = path.split('/').filter(s => s.length > 0);
        return segments[segments.length - 1] || 'homepage';
    }

    function getPostIdFromMeta() {
        const meta = document.querySelector('meta[name="post-id"]');
        return meta ? meta.getAttribute('content') : null;
    }

    function getPostIdFromTitle() {
        const title = document.title;
        // Einfache Slugify-Funktion
        return title.toLowerCase()
            .replace(/[^a-z0-9 -]/g, '') // Sonderzeichen entfernen
            .replace(/\s+/g, '-')        // Leerzeichen zu Bindestrichen
            .replace(/-+/g, '-')         // Mehrfache Bindestriche reduzieren
            .trim();
    }

    // Automatische Post-ID-Erkennung
    function detectPostId() {
        // Prioritätenreihenfolge:
        // 1. Meta-Tag
        // 2. URL-basiert
        // 3. Title-basiert
        return getPostIdFromMeta() || getPostIdFromUrl() || getPostIdFromTitle();
    }

    // Nachricht anzeigen
    function showMessage(container, message, type) {
        const messageContainer = container.querySelector('.comment-message-container');
        messageContainer.innerHTML = `<div class="comment-message ${type}">${message}</div>`;
        setTimeout(() => {
            messageContainer.innerHTML = '';
        }, 5000);
    }

    // HTML escaping
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Kommentare laden
    async function loadComments(postId, container) {
        const commentsContainer = container.querySelector('.comments-container');
        
        try {
            const response = await fetch(`${config.apiUrl}?post_id=${encodeURIComponent(postId)}`);
            
            if (response.ok) {
                const comments = await response.json();
                displayComments(comments, commentsContainer);
            } else {
                commentsContainer.innerHTML = '<div class="comments-loading">Fehler beim Laden der Kommentare</div>';
            }
        } catch (error) {
            console.error('Fehler beim Laden der Kommentare:', error);
            commentsContainer.innerHTML = '<div class="comments-loading">Verbindungsfehler</div>';
        }
    }

    // Kommentare anzeigen
    function displayComments(comments, container) {
        if (!comments || comments.length === 0) {
            container.innerHTML = '<div class="comments-empty">Noch keine Kommentare vorhanden. Sei der erste! 🚀</div>';
            return;
        }

        comments.sort((b, a) => new Date(b.created_at) - new Date(a.created_at));

        const commentsHTML = comments.map(comment => {
            const date = new Date(comment.created_at).toLocaleDateString('de-DE', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });

            return `
                <div class="comment-item">
                    <div class="comment-header">
                        <span class="comment-author">${escapeHtml(comment.username)}</span>
                        <span class="comment-date">${date}</span>
                    </div>
                    <div class="comment-text">${escapeHtml(comment.text)}</div>
                </div>
            `;
        }).join('');

        container.innerHTML = commentsHTML;
    }

    // Kommentar absenden
    async function submitComment(postId, formData, container) {
        const submitBtn = container.querySelector('.comment-submit-btn');
        const form = container.querySelector('.comment-form-element');
        
        submitBtn.disabled = true;
        submitBtn.textContent = 'Wird gesendet...';

        const commentData = {
            post_id: postId,
            username: formData.get('username'),
            mailaddress: formData.get('mailaddress'),
            text: formData.get('text')
        };

        try {
            const response = await fetch(config.apiUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(commentData)
            });

            if (response.ok) {
                showMessage(container, 'Kommentar erfolgreich erstellt! 🎉', 'success');
                form.reset();
                loadComments(postId, container);
            } else {
                const errorText = await response.text();
                showMessage(container, `Fehler: ${errorText}`, 'error');
            }
        } catch (error) {
            console.error('Fehler beim Absenden:', error);
            showMessage(container, 'Verbindungsfehler. Bitte versuche es später erneut.', 'error');
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Kommentar absenden';
        }
    }

    // Widget an einem Element initialisieren
    function initWidget(postId, targetElement, options = {}) {
        // Konfiguration überschreiben
        const localConfig = { ...config, ...options };
        
        // CSS injizieren
        injectCSS();

        // Widget HTML erstellen und einfügen
        targetElement.innerHTML = createWidgetHTML(postId);
        const widget = targetElement.querySelector('.comment-widget');

        // Event-Listener für das Formular
        const form = widget.querySelector('.comment-form-element');
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(form);
            await submitComment(postId, formData, widget);
        });

        // Kommentare initial laden
        loadComments(postId, widget);

        // Auto-refresh alle 60 Sekunden
        setInterval(() => {
            loadComments(postId, widget);
        }, 60000);

        return widget;
    }

    // Manueller Init (wie bisher)
    function init(postId, options = {}, targetElement = null) {
        if (targetElement) {
            return initWidget(postId, targetElement, options);
        }

        // Legacy-Verhalten: Am aktuellen Script einfügen
        const currentScript = document.currentScript;
        const container = document.createElement('div');
        
        if (currentScript && currentScript.parentNode) {
            currentScript.parentNode.insertBefore(container, currentScript.nextSibling);
        } else {
            document.body.appendChild(container);
        }
        
        return initWidget(postId, container, options);
    }

    // Automatische Initialisierung für data-Attribute
    function autoInit(options = {}) {
        const elements = document.querySelectorAll('[data-comment-post-id]');
        const widgets = [];
        
        elements.forEach(element => {
            const postId = element.getAttribute('data-comment-post-id');
            if (postId && !element.querySelector('.comment-widget')) {
                const widget = initWidget(postId, element, options);
                widgets.push(widget);
            }
        });
        
        return widgets;
    }

    // Auto-Init basierend auf URL
    function autoInitFromUrl(options = {}) {
        const postId = detectPostId();
        const targetElement = document.getElementById('comments') || 
                             document.querySelector('.comments') ||
                             document.querySelector('[data-comments]');
        
        if (targetElement) {
            return initWidget(postId, targetElement, options);
        } else {
            console.warn('CommentWidget: Kein Target-Element für Auto-Init gefunden');
            return null;
        }
    }

    // Konfiguration setzen
    function configure(newConfig) {
        config = { ...config, ...newConfig };
    }

    // Auto-Init beim DOM-Laden
    function setupAutoInit() {
        const runAutoInit = () => {
            autoInit();
        };

        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', runAutoInit);
        } else {
            runAutoInit();
        }
    }

    // Setup ausführen
    setupAutoInit();

    // Öffentliche API
    return {
        init: init,
        autoInit: autoInit,
        autoInitFromUrl: autoInitFromUrl,
        configure: configure,
        config: config
    };
})();