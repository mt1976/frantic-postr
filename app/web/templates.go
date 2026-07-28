package web

import "html/template"

var webIndexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.AppName}} Web UI</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/css/uikit.min.css">
  <link rel="manifest" href="/res/site.webmanifest">
  <link rel="icon" href="/res/favicon.svg" type="image/svg+xml">
  <link rel="apple-touch-icon" href="/res/icon-192.svg">
  <meta name="theme-color" content="#23353c">
  <link rel="stylesheet" href="/res/app.css">
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit.min.js"></script>
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit-icons.min.js"></script>
</head>
<body>
  <div class="fp-shell">
    <section class="fp-hero uk-margin-large-bottom uk-animation-slide-top-small">
      <div class="uk-flex uk-flex-between uk-flex-wrap uk-flex-middle uk-gap-small">
        <div>
          <div class="fp-kicker">{{.Mode}} <span>•</span> Version {{.Version}}</div>
          <h1 class="fp-title uk-heading-small">{{.AppName}} control room</h1>
		  <p class="uk-text-large uk-margin-small-top uk-margin-medium-bottom fp-hero-description">Run poster generation, library cleanup, collection maintenance, and backup workflows from a local UIKit dashboard without losing the existing Plex-aware Go logic.</p>
		  <div class="uk-flex uk-flex-wrap uk-gap-small">
						<button class="uk-button uk-button-secondary" id="refresh-state"><span uk-icon="refresh" class="uk-margin-small-right"></span>Refresh state</button>
						<button class="uk-button uk-button-default" type="button" uk-toggle="target: #help-modal"><span uk-icon="question" class="uk-margin-small-right"></span>Help & tips</button>
						<button class="uk-button uk-button-default" uk-toggle="target: #about-modal"><span uk-icon="info" class="uk-margin-small-right"></span>About</button>
          </div>
        </div>
		<div class="fp-summary-grid fp-summary-grid-wide">
          <div class="fp-stat">
            <span class="fp-kicker">Listening</span>
						<strong>0.0.0.0:{{.Port}}</strong>
          </div>
          <div class="fp-stat">
            <span class="fp-kicker">Started</span>
            <strong>{{.StartedAt}}</strong>
          </div>
        </div>
      </div>
    </section>

    <div id="status-banner" class="fp-banner uk-margin-bottom"></div>

	<div class="uk-width-expand@l">
		<div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-gap-small">
			<ul uk-tab="connect: #main-switcher" class="uk-margin-remove-bottom">
				<li><a href="#"><span uk-icon="settings" class="uk-margin-small-right"></span>Config</a></li>
				<li><a href="#"><span uk-icon="cog" class="uk-margin-small-right"></span>Runtime</a></li>
				<li><a href="#"><span uk-icon="image" class="uk-margin-small-right"></span>Posters</a></li>
				<li><a href="#"><span uk-icon="folder" class="uk-margin-small-right"></span>Library</a></li>
				<li><a href="#"><span uk-icon="thumbnails" class="uk-margin-small-right"></span>Collections</a></li>
				<li><a href="#"><span uk-icon="lock" class="uk-margin-small-right"></span>Backup & Restore</a></li>
			</ul>
			<div>
				<button class="uk-button uk-button-primary fp-hidden" id="save-config-global" type="submit" form="config-form"><span class="uk-margin-small-right" aria-hidden="true">&#128190;</span>Save config</button>
			</div>
		</div>
		<ul id="main-switcher" class="uk-switcher uk-margin">
			<li>
				<form id="config-form">
					<div class="fp-actions-grid">
						<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
							<h2 class="fp-section-title">Configuration</h2>
							<p class="fp-muted uk-margin-small-top">If you do not pass <code>--config</code>, the web UI uses <code>config/config.toml</code>. Saving here writes the exposed settings back to the active config files, and keeps Plex secrets in <code>plex_config</code> when you use a split file.</p>
							<div class="uk-grid-small" uk-grid>
								<div class="uk-width-1-1">
									<label class="uk-form-label" for="output-dir-input">Output directory</label>
									<input class="uk-input" id="output-dir-input" name="output_dir" type="text">
								</div>
								<div class="uk-width-1-1">
									<label class="uk-form-label" for="log-file-input">Log file</label>
									<input class="uk-input" id="log-file-input" name="log_file" type="text">
								</div>
								<div class="uk-width-1-2">
									<label class="uk-form-label" for="backup-retention-days">Backup retention days</label>
									<input class="uk-input" id="backup-retention-days" type="number" min="0">
								</div>
								<div class="uk-width-1-1">
									<p class="fp-footer-note uk-margin-remove-bottom">The server stays local to 127.0.0.1.</p>
								</div>
							</div>
						</div>
						<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
							<h2 class="fp-section-title">Plex Configuration</h2>
							<div class="uk-grid-small" uk-grid>
								<div class="uk-width-1-1">
									<label class="uk-form-label" for="base-url">Plex URL</label>
									<input class="uk-input" id="base-url" name="base_url" type="url" required>
								</div>
								<div class="uk-width-1-1">
									<label class="uk-form-label" for="token">API key / token</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<input class="uk-input fp-min-260" id="token" name="token" type="password" autocomplete="off">
										<button class="uk-button uk-button-default" type="button" data-toggle-password="token" aria-label="Reveal token"><span uk-icon="eye" class="uk-margin-small-right"></span>Show</button>
									</div>
								</div>
								<div class="uk-width-1-2">
									<label class="uk-form-label" for="retries">Retries</label>
									<input class="uk-input" id="retries" name="retries" type="number" min="1">
								</div>
								<div class="uk-width-1-2">
									<label class="uk-form-label" for="workers">Workers</label>
									<input class="uk-input" id="workers" name="workers" type="number" min="1">
								</div>
								<div class="uk-width-1-2">
									<label class="uk-form-label" for="retry-base-ms">Retry base ms</label>
									<input class="uk-input" id="retry-base-ms" name="retry_base_ms" type="number" min="1">
								</div>
								<div class="uk-width-1-2">
									<label class="uk-form-label" for="retry-max-ms">Retry max ms</label>
									<input class="uk-input" id="retry-max-ms" name="retry_max_ms" type="number" min="1">
								</div>
								<div class="uk-width-1-1">
									<button class="uk-button uk-button-secondary fp-test-button" id="test-plex-connection" type="button"><span class="fp-glyph-icon uk-margin-small-right" aria-hidden="true">&#129514;</span>Test Plex connection <span class="fp-test-button-icon" id="test-plex-connection-icon" aria-hidden="true"></span></button>
								</div>
							</div>
						</div>
						<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
							<h2 class="fp-section-title">Config Files and Templates</h2>
							<p class="fp-muted uk-margin-small-top">Template images can be selected from uploaded files or replaced by uploading a new image.</p>
							<div class="uk-grid-small" uk-grid>
								<div class="uk-width-1-1 fp-template-manager" data-target-input="template-image" data-upload-role="template-image-main" data-label="Template image">
									<label class="uk-form-label" for="template-image-select">Template image</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select fp-template-select fp-min-260" id="template-image-select"></select>
										<input class="uk-input fp-template-upload fp-min-260" id="template-image-upload" type="file" accept=".png,.jpg,.jpeg,.webp,.gif,.bmp,.tif,.tiff,.avif">
										<button class="uk-button uk-button-secondary fp-template-upload-btn" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload</button>
									</div>
								</div>
								<div class="uk-width-1-1 fp-template-manager" data-target-input="type-template-image" data-upload-role="template-image-type" data-label="Type template image">
									<label class="uk-form-label" for="type-template-image-select">Type template image</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select fp-template-select fp-min-260" id="type-template-image-select"></select>
										<input class="uk-input fp-template-upload fp-min-260" id="type-template-image-upload" type="file" accept=".png,.jpg,.jpeg,.webp,.gif,.bmp,.tif,.tiff,.avif">
										<button class="uk-button uk-button-secondary fp-template-upload-btn" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload</button>
									</div>
								</div>
								<div class="uk-width-1-1 fp-template-manager" data-target-input="studio-template-image" data-upload-role="template-image-studio" data-label="Studio template image">
									<label class="uk-form-label" for="studio-template-image-select">Studio template image</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select fp-template-select fp-min-260" id="studio-template-image-select"></select>
										<input class="uk-input fp-template-upload fp-min-260" id="studio-template-image-upload" type="file" accept=".png,.jpg,.jpeg,.webp,.gif,.bmp,.tif,.tiff,.avif">
										<button class="uk-button uk-button-secondary fp-template-upload-btn" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload</button>
									</div>
								</div>
								<div class="uk-width-1-1 fp-template-manager" data-target-input="admin-template-image" data-upload-role="template-image-admin" data-label="Admin template image">
									<label class="uk-form-label" for="admin-template-image-select">Admin template image</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select fp-template-select fp-min-260" id="admin-template-image-select"></select>
										<input class="uk-input fp-template-upload fp-min-260" id="admin-template-image-upload" type="file" accept=".png,.jpg,.jpeg,.webp,.gif,.bmp,.tif,.tiff,.avif">
										<button class="uk-button uk-button-secondary fp-template-upload-btn" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload</button>
									</div>
								</div>
								<div class="uk-width-1-1">
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<button class="uk-button uk-button-default" type="button" data-config-editor="type-collections"><span uk-icon="list" class="uk-margin-small-right"></span>Edit type collections list</button>
										<button class="uk-button uk-button-default" type="button" data-config-editor="studio-collections"><span uk-icon="list" class="uk-margin-small-right"></span>Edit studio collections list</button>
										<button class="uk-button uk-button-default" type="button" data-config-editor="admin-collections"><span uk-icon="list" class="uk-margin-small-right"></span>Edit admin collections list</button>
									</div>
								</div>
								<div class="uk-width-1-1">
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<button class="uk-button uk-button-default" type="button" data-config-editor="label-config"><span uk-icon="file-edit" class="uk-margin-small-right"></span>Edit label config</button>
										<button class="uk-button uk-button-default" type="button" data-config-editor="collection-config"><span uk-icon="file-edit" class="uk-margin-small-right"></span>Edit collection config</button>
									</div>
									<p class="fp-footer-note uk-margin-small-top">Plex supplemental config path is intentionally hidden in web mode.</p>
								</div>
							</div>
							<input id="template-image" name="template_image" type="hidden">
							<input id="type-template-image" name="type_template_image" type="hidden">
							<input id="studio-template-image" name="studio_template_image" type="hidden">
							<input id="admin-template-image" name="admin_template_image" type="hidden">
							<input id="type-collections-file" name="type_collections_file" type="hidden">
							<input id="studio-collections-file" name="studio_collections_file" type="hidden">
							<input id="admin-collections-file" name="admin_collections_file" type="hidden">
							<input id="plex-config-file" name="plex_config" type="hidden">
							<input id="label-config-file" name="label_config" type="hidden">
							<input id="collection-config-file" name="collection_config" type="hidden">
						</div>
					<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
						<h2 class="fp-section-title">Poster Configuration</h2>
						<p class="fp-muted uk-margin-small-top">Generate a sample render on demand to preview how text styling appears on each template type.</p>
						<div class="uk-grid-small" uk-grid>
							<div class="uk-width-1-1">
								<label class="uk-form-label" for="preview-template-kind">Template type</label>
								<select id="preview-template-kind" class="uk-select">
									<option value="default">Default template</option>
									<option value="type">Type template</option>
									<option value="studio">Studio template</option>
									<option value="admin">Admin template</option>
								</select>
							</div>
							<div class="uk-width-1-1">
								<label class="uk-form-label" for="preview-sample-text">Sample text</label>
								<input id="preview-sample-text" class="uk-input" type="text" value="Sample Collection 2026" placeholder="Sample Collection 2026">
							</div>
							<div class="uk-width-1-1">
								<button class="uk-button uk-button-secondary" id="generate-template-preview" type="button"><span uk-icon="image" class="uk-margin-small-right"></span>Generate sample</button>
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-file">Font file</label>
								<input class="uk-input" id="font-file" type="text">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-size">Font size</label>
								<input class="uk-input" id="font-size" type="number" min="0" step="0.1">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-color">Font color</label>
								<input class="uk-input" id="font-color" type="text" placeholder="#FFFFFF">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-shadow-color">Shadow color</label>
								<input class="uk-input" id="font-shadow-color" type="text">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-glow-color">Glow color</label>
								<input class="uk-input" id="font-glow-color" type="text">
							</div>
							<div class="uk-width-1-3">
								<label class="uk-form-label" for="font-shadow-offset-x">Shadow X</label>
								<input class="uk-input" id="font-shadow-offset-x" type="number">
							</div>
							<div class="uk-width-1-3">
								<label class="uk-form-label" for="font-shadow-offset-y">Shadow Y</label>
								<input class="uk-input" id="font-shadow-offset-y" type="number">
							</div>
							<div class="uk-width-1-3">
								<label class="uk-form-label" for="font-glow-radius">Glow radius</label>
								<input class="uk-input" id="font-glow-radius" type="number" min="0">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-glow-alpha">Glow alpha</label>
								<input class="uk-input" id="font-glow-alpha" type="number" min="0" max="1" step="0.01">
							</div>
							<div class="uk-width-1-2">
								<label class="uk-form-label" for="font-y-offset">Y offset</label>
								<input class="uk-input" id="font-y-offset" type="number">
							</div>
						</div>
						<div id="template-preview-meta" class="fp-footer-note uk-margin-small-top">No preview generated yet.</div>
						<div id="template-preview-panel" class="fp-preview-panel uk-margin-small-top">
							<img id="template-preview-image" class="fp-preview-image" alt="Template preview">
							<p id="template-preview-empty" class="fp-muted uk-margin-remove">Click Generate sample to render a preview.</p>
						</div>
					</div>
					<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
						<h2 class="fp-section-title">Text Cleanup and Stats</h2>
						<p class="fp-muted uk-margin-small-top">Manage translation, clean replacements, and stats exclude words with modern editors and raw fallbacks.</p>
						<div class="uk-grid-small" uk-grid>
							<div class="uk-width-1-1">
								<label class="uk-form-label" for="translate-endpoint">Translate endpoint</label>
								<input class="uk-input" id="translate-endpoint" name="translate_endpoint" type="url">
							</div>
							<div class="uk-width-1-1">
								<label class="uk-form-label" for="translate-api-key">Translate API key</label>
								<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
									<input class="uk-input fp-min-260" id="translate-api-key" name="translate_api_key" type="password" autocomplete="off">
									<button class="uk-button uk-button-default" type="button" data-toggle-password="translate-api-key" aria-label="Reveal translate API key"><span uk-icon="eye" class="uk-margin-small-right"></span>Show</button>
								</div>
							</div>
							<div class="uk-width-1-1">
								<label class="uk-form-label" for="translate-rate-limit">Translate rate limit per minute</label>
								<input class="uk-input" id="translate-rate-limit" name="translate_rate_limit_per_minute" type="number" min="1">
							</div>
							<div class="uk-width-1-1">
								<label><input class="uk-checkbox" id="translate-to-english" type="checkbox"> Translate to English in clean config</label>
							</div>
							<div class="uk-width-1-1">
								<label class="uk-form-label">Clean replacements</label>
								<div class="fp-inline-editor">
									<div class="uk-grid-small" uk-grid>
										<div class="uk-width-2-5@s"><input class="uk-input" id="clean-repl-find" type="text" placeholder="Find text"></div>
										<div class="uk-width-2-5@s"><input class="uk-input" id="clean-repl-replace" type="text" placeholder="Replace with"></div>
										<div class="uk-width-1-5@s"><button class="uk-button uk-button-secondary uk-width-1-1" id="clean-repl-add" type="button">Add</button></div>
									</div>
									<div id="clean-repl-list" class="fp-config-list uk-margin-small-top"></div>
									<details class="uk-margin-small-top">
										<summary class="fp-footer-note">Raw editor</summary>
										<textarea class="uk-textarea uk-margin-small-top" id="clean-replacements" rows="6" placeholder="& = and\nFULL MOVIE = "></textarea>
									</details>
								</div>
							</div>
							<div class="uk-width-1-1">
								<label class="uk-form-label">Stats exclude words</label>
								<div class="fp-inline-editor">
									<div class="uk-grid-small" uk-grid>
										<div class="uk-width-4-5@s uk-width-1-1"><input class="uk-input" id="stats-word-input" type="text" placeholder="Add word"></div>
										<div class="uk-width-1-5@s uk-width-1-1"><button class="uk-button uk-button-secondary uk-width-1-1" id="stats-word-add" type="button">Add</button></div>
									</div>
									<div id="stats-words-chip-list" class="fp-inline-chip-wrap"></div>
									<details class="uk-margin-small-top">
										<summary class="fp-footer-note">Raw editor</summary>
										<textarea class="uk-textarea uk-margin-small-top" id="stats-exclude-words" rows="4" placeholder="comma,separated,words"></textarea>
									</details>
								</div>
							</div>
						</div>
					</div>
					</div>
				</form>
			</li>
			<li>
				<div class="fp-actions-grid">
					<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
						<h2 class="fp-section-title">Runtime</h2>
						<dl class="uk-description-list uk-description-list-divider">
							<dt>Config</dt>
							<dd id="config-path" class="fp-path"></dd>
							<dt>Output</dt>
							<dd id="output-dir" class="fp-path"></dd>
							<dt>Logs</dt>
							<dd id="log-file" class="fp-path"></dd>
							<dt>Backups</dt>
							<dd id="backup-list"></dd>
						</dl>
					</div>
				</div>
			</li>
		  <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Generate posters</h3>
                <form id="posters-form">
                  <label class="uk-form-label" for="poster-sections">Libraries</label>
                  <select class="uk-select" id="poster-sections" multiple size="8"></select>
                  <div class="uk-margin-small-top uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="poster-upload" type="checkbox"> Upload posters to Plex</label>
                    <label><input class="uk-checkbox" id="poster-label-types" type="checkbox"> Label type collections</label>
										<label><input class="uk-checkbox" id="poster-missing-only" type="checkbox"> Only collections missing dedicated posters</label>
                    <label><input class="uk-checkbox" id="poster-trail" type="checkbox"> Dry run only</label>
                  </div>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run poster workflow</button>
                </form>
              </div>
            </div>
          </li>
		  <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Clean titles</h3>
                <form id="clean-form">
                  <label class="uk-form-label" for="clean-section">Library</label>
                  <select class="uk-select" id="clean-section"></select>
                  <div class="uk-margin-small-top uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="clean-translate" type="checkbox"> Translate before cleaning</label>
                    <label><input class="uk-checkbox" id="clean-trail" type="checkbox"> Dry run only</label>
                  </div>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run clean</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Translate titles</h3>
                <form id="translate-form">
                  <label class="uk-form-label" for="translate-section">Library</label>
                  <select class="uk-select" id="translate-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="translate-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run translation</button>
									</div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Stats and labels</h3>
                <form id="stats-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="stats-section">Stats library</label>
                  <select class="uk-select" id="stats-section"></select>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run stats</button>
                </form>
                <hr>
                <form id="label-form" class="uk-grid-small" uk-grid>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-section">Label library</label>
                    <select class="uk-select" id="label-section"></select>
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-find">Find text</label>
                    <input class="uk-input" id="label-find" type="text" placeholder="studio, actor, tag fragment">
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-add">Labels</label>
                    <input class="uk-input" id="label-add" type="text" placeholder="comma,separated,labels">
                  </div>
                  <div class="uk-width-1-1">
                    <label class="uk-form-label" for="label-categories">Categories</label>
                    <input class="uk-input" id="label-categories" type="text" placeholder="optional override">
                  </div>
                  <div class="uk-width-1-1 uk-grid-small" uk-grid>
                    <label><input class="uk-checkbox" id="label-update-category" type="checkbox"> Update categories</label>
                    <label><input class="uk-checkbox" id="label-only-category" type="checkbox"> Categories only</label>
                    <label><input class="uk-checkbox" id="label-trail" type="checkbox"> Dry run only</label>
                  </div>
                  <div class="uk-width-1-1">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run labels</button>
                  </div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Clone library</h3>
                <form id="clone-form">
                  <label class="uk-form-label" for="clone-section">Source library</label>
                  <select class="uk-select" id="clone-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="clone-name">New library name</label>
                  <input class="uk-input" id="clone-name" type="text" placeholder="movies-clone">
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="copy" class="uk-margin-small-right"></span>Clone</button>
                </form>
              </div>
            </div>
          </li>
		  <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Export and import</h3>
                <form id="export-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="export-section">Export library</label>
                  <select class="uk-select" id="export-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="export-file">Export file</label>
                  <input class="uk-input" id="export-file" type="text">
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="download" class="uk-margin-small-right"></span>Export collections</button>
                </form>
                <hr>
                <form id="import-form">
                  <label class="uk-form-label" for="import-section">Import target library</label>
                  <select class="uk-select" id="import-section"></select>
									<label class="uk-form-label uk-margin-small-top" for="import-file">Import file</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select fp-min-320" id="import-file"></select>
										<button class="uk-button uk-button-default" id="refresh-import-files" type="button"><span uk-icon="refresh" class="uk-margin-small-right"></span>Refresh files</button>
									</div>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle uk-margin-small-top">
										<input class="uk-input fp-min-320" id="import-file-upload" type="file" accept=".json,application/json">
										<button class="uk-button uk-button-secondary" id="upload-import-file" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload import file</button>
									</div>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="import-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="upload" class="uk-margin-small-right"></span>Import collections</button>
									</div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Collection maintenance</h3>
                <form id="inject-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="inject-section">Inject target library</label>
                  <select class="uk-select" id="inject-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="inject-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="plus" class="uk-margin-small-right"></span>Inject configured collections</button>
									</div>
                </form>
                <form id="dupes-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="dupes-section">Duplicates library</label>
                  <select class="uk-select" id="dupes-section"></select>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="search" class="uk-margin-small-right"></span>Audit duplicates</button>
                </form>
                <form id="delete-non-smart-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="delete-non-smart-section">Delete non-smart library</label>
                  <select class="uk-select" id="delete-non-smart-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="delete-non-smart-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-danger" type="submit"><span uk-icon="trash" class="uk-margin-small-right"></span>Delete non-smart collections</button>
									</div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Path clean a collection</h3>
                <form id="path-clean-form">
                  <label class="uk-form-label" for="path-clean-section">Library</label>
                  <select class="uk-select" id="path-clean-section"></select>
                  <label class="uk-form-label uk-margin-small-top" for="path-clean-collection">Collection</label>
                  <select class="uk-select" id="path-clean-collection"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="path-clean-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Path clean collection</button>
									</div>
                </form>
              </div>
            </div>
          </li>
		  <li>
            <div class="fp-actions-grid">
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Backup</h3>
                <p class="fp-muted">Archive config, templates, fonts, and selection state into the repo-level backups folder.</p>
                <form id="backup-form">
									<button class="uk-button uk-button-primary" type="submit"><span uk-icon="database" class="uk-margin-small-right"></span>Create backup</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Restore</h3>
                <p class="fp-muted">Leave blank to restore the newest backup, or paste part of a filename to target a specific archive.</p>
                <form id="restore-form">
                  <label class="uk-form-label" for="restore-file">Backup filter</label>
                  <input class="uk-input" id="restore-file" type="text" placeholder="20260704 or frantic-postr-backup-host">
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="restore-trail" type="checkbox"> Dry run only</label>
									<div class="fp-action-row uk-margin-small-top">
										<button class="uk-button uk-button-primary" type="submit"><span uk-icon="history" class="uk-margin-small-right"></span>Restore backup</button>
									</div>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Rollback</h3>
                <p class="fp-muted">Revert the most recent restore run using the generated rollback archive and manifest.</p>
                <form id="rollback-form">
									<button class="uk-button uk-button-danger" type="submit"><span uk-icon="reply" class="uk-margin-small-right"></span>Rollback last restore</button>
                </form>
              </div>
            </div>
          </li>
		</ul>

		<div class="uk-card uk-card-body fp-card uk-margin-top uk-animation-slide-bottom-small">
			<div class="uk-flex uk-flex-between uk-flex-wrap uk-flex-middle uk-gap-small">
				<div>
					<h2 class="fp-section-title">Operation progress</h2>
					<p class="fp-muted uk-margin-small-top">Progress stays visible while you switch tabs, useful for remote and Docker-based runs.</p>
				</div>
				<div class="uk-flex uk-flex-wrap uk-gap-small uk-flex-middle fp-button-group">
					<button class="uk-button uk-button-danger uk-button-small" id="stop-action" type="button" disabled><span uk-icon="ban" class="uk-margin-small-right"></span>Stop active process</button>
					<button class="uk-button uk-button-secondary uk-button-small" id="download-output" type="button" disabled>Download output file</button>
					<span class="fp-chip" id="config-status-chip">Loading runtime status…</span>
				</div>
			</div>
			<div class="fp-progress-wrap uk-margin-top">
				<div class="fp-progress-head">
					<p class="fp-progress-title" id="progress-label">No active operation</p>
					<span id="progress-count">0/0</span>
				</div>
				<progress id="progress-bar" class="fp-progress-bar" max="100" value="0"></progress>
				<div class="fp-progress-meta" id="progress-meta">Idle</div>
			</div>
		</div>

		<ul uk-accordion class="uk-margin-top">
			<li>
				<a class="uk-accordion-title" href="#"><span uk-icon="file-text" class="uk-margin-small-right"></span>Operation log</a>
				<div class="uk-accordion-content">
					<p class="fp-muted uk-margin-small-top">Action output is always available here, no matter which tab is active.</p>
					<div class="fp-log-wrap uk-margin-top">
						<button class="uk-button uk-button-default uk-button-small fp-copy-log" id="copy-log" type="button"><span uk-icon="copy" class="uk-margin-small-right"></span>Copy</button>
						<pre id="action-log" class="fp-log">No action has been run yet.</pre>
					</div>
				</div>
			</li>
		</ul>
	</div>
  </div>

	<div id="config-editor-modal" uk-modal>
		<div class="uk-modal-dialog uk-modal-body fp-card">
			<button class="uk-modal-close-default" type="button" uk-close></button>
			<h3 id="config-editor-title" class="fp-section-title">Edit config</h3>
			<p id="config-editor-path" class="fp-muted uk-margin-small-top"></p>
			<div id="config-editor-mode-switch" class="fp-editor-mode-switch" hidden>
				<button class="uk-button uk-button-small uk-button-primary" id="config-editor-mode-structured" type="button">Structured</button>
				<button class="uk-button uk-button-small uk-button-default" id="config-editor-mode-raw" type="button">Raw text</button>
			</div>
			<div id="config-editor-list-mode" hidden>
				<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle uk-margin-small-bottom">
						<input id="config-editor-item-input" class="uk-input fp-min-280" type="text" placeholder="Add new entry">
					<button class="uk-button uk-button-primary" id="config-editor-add-item" type="button"><span uk-icon="plus" class="uk-margin-small-right"></span>Add</button>
				</div>
				<div id="config-editor-list" class="fp-config-list"></div>
			</div>
			<div id="config-editor-structured" class="fp-config-structured" hidden></div>
			<textarea id="config-editor-text" class="uk-textarea fp-mono" rows="14"></textarea>
			<div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-margin-top uk-gap-small">
				<span id="config-editor-hint" class="fp-footer-note">Edit and save.</span>
				<div class="uk-flex uk-gap-small uk-flex-wrap">
					<button class="uk-button uk-button-default" id="config-editor-reload" type="button">Reload</button>
					<button class="uk-button uk-button-primary" id="config-editor-save" type="button"><span uk-icon="check" class="uk-margin-small-right"></span>Save changes</button>
				</div>
			</div>
		</div>
	</div>

	<div id="help-modal" uk-modal>
		<div class="uk-modal-dialog uk-modal-body fp-card fp-help-modal-dialog">
			<button class="uk-modal-close-default" type="button" uk-close></button>
			<div class="fp-help-modal-scroll">
				<div class="fp-help-card">
					<div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-gap-small">
						<div>
						<p class="uk-text-meta fp-help-kicker">Useful tips</p>
							<h2 class="uk-modal-title uk-margin-small-top">{{.AppName}} web help</h2>
						</div>
					</div>
					<p class="uk-text-large">The web UI is local-only, uses the same Go workflows as the CLI, and is best treated as an operations console rather than a separate product tier.</p>
				</div>

				<div class="fp-help-card">
					<h3>Recommended flow</h3>
					<ul class="uk-list uk-list-bullet">
						<li>Confirm the Plex URL and token first. A valid config unlocks library discovery and all Plex-backed actions.</li>
						<li>Use dry-run checkboxes before destructive actions such as cleaning titles, deleting non-smart collections, imports, and restores.</li>
						<li>Run stats or duplicate-collection audits before cleanup so you have fresh reports in the output folders.</li>
						<li>Use backup before large operations that touch multiple files or collections.</li>
					</ul>
				</div>

				<div class="fp-help-card">
					<h3>Action notes</h3>
					<ul class="uk-list uk-list-bullet">
						<li><code>Generate posters</code> can target multiple libraries at once and optionally upload finished posters back to Plex.</li>
						<li><code>Clean titles</code> can optionally translate titles first, then apply the configured cleanup replacements.</li>
						<li><code>Path clean</code> rewrites titles from collection item file paths and writes a CSV audit into <code>output/path-clean/</code>.</li>
						<li><code>Export collections</code> writes JSON under <code>output/collections-export/</code> when you provide only a filename.</li>
						<li><code>Restore</code> accepts a partial backup filename; leaving it blank restores the newest archive in non-interactive mode.</li>
					</ul>
				</div>

				<div class="fp-help-card">
					<h3>Troubleshooting</h3>
					<ul class="uk-list uk-list-bullet">
						<li>If the dashboard loads but libraries do not, the config is readable but the Plex connection is failing. Check URL, token, and network reachability.</li>
						<li>If an operation says config validation failed, fix missing template, font, or output paths before retrying that workflow.</li>
						<li>If a restore or rollback fails, review the operation log first, then inspect the latest files under <code>output/restore/</code>.</li>
					</ul>
				</div>
			</div>
			<div class="uk-flex uk-flex-right uk-margin-top">
				<button class="uk-button uk-button-primary uk-modal-close" type="button">Close</button>
			</div>
		</div>
	</div>

  <div id="about-modal" uk-modal>
    <div class="uk-modal-dialog uk-modal-body fp-card">
      <button class="uk-modal-close-default" type="button" uk-close></button>
      <h2 class="uk-modal-title">About {{.AppName}}</h2>
      <p class="fp-muted">A local control surface for the existing frantic-postr Go workflows. The UI stays on top of the same Plex-aware code paths used by the CLI.</p>
      <dl class="uk-description-list uk-description-list-divider">
        <dt>Version</dt>
        <dd>{{.About.Version}}</dd>
        <dt>Go runtime</dt>
        <dd>{{.About.GoVersion}}</dd>
        <dt>Module</dt>
        <dd>{{.About.Module}}</dd>
        <dt>Mode</dt>
        <dd>{{.About.Mode}}</dd>
        <dt>Started</dt>
        <dd>{{.About.StartedAt}}</dd>
        {{if .About.CommitHint}}<dt>Build hint</dt><dd>{{.About.CommitHint}}</dd>{{end}}
      </dl>
    </div>
  </div>

  <script>
	const state = {
		sections: [],
		backups: [],
		defaults: {},
		templateFiles: [],
		plexTestResetTimer: null,
		actionPollTimer: null,
		actionPollBusy: false,
		actionPollFast: 300,
		actionPollSlow: 1200,
		lastActionStatusHash: '',
		lastDownloadToken: '',
		runtimeStatusToastShown: false,
		lastParsedLogText: '',
		seenToastLogLines: {},
		configEditorScope: '',
		configEditorTitle: '',
		configEditorItems: [],
		statsExcludeWordsItems: [],
		cleanReplacementItems: [],
		configEditorRawContent: '',
		configEditorStructuredContent: '',
		configEditorMode: 'structured',
		configEditorStructuredReady: false
	};

    const sectionSelectIds = [
      'clean-section', 'translate-section', 'stats-section', 'label-section', 'clone-section',
      'export-section', 'import-section', 'inject-section', 'dupes-section', 'delete-non-smart-section',
      'path-clean-section'
    ];

    function showBanner(type, message) {
      const banner = document.getElementById('status-banner');
	banner.className = 'fp-banner active ' + type;
      banner.textContent = message;
    }

		function hideBanner() {
			const banner = document.getElementById('status-banner');
			if (!banner) {
				return;
			}
			banner.className = 'fp-banner';
			banner.textContent = '';
		}

		function showToast(type, message, timeout) {
			if (typeof UIkit === 'undefined' || !UIkit.notification) {
				return;
			}
			const status = type === 'error' ? 'danger' : type;
			UIkit.notification({
				message: message,
				status: status,
				pos: 'top-right',
				timeout: timeout || 4000
			});
		}

		function setPasswordVisible(inputId, visible, button) {
			const input = document.getElementById(inputId);
			if (!input || !button) {
				return;
			}
			input.type = visible ? 'text' : 'password';
			button.innerHTML = '<span uk-icon="' + (visible ? 'eye-slash' : 'eye') + '" class="uk-margin-small-right"></span>' + (visible ? 'Hide' : 'Show');
			if (typeof UIkit !== 'undefined' && UIkit.icon) {
				UIkit.icon(button.querySelector('[uk-icon]'));
			}
		}

		function sanitizeLogText(text) {
			const raw = text || '';
			const withoutAnsi = raw
				.replace(/\u001b\[[0-9;]*[ -\/]*[@-~]/g, '')
				.replace(/\u001b\][^\u0007]*(\u0007|\u001b\\)/g, '');
			return withoutAnsi.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '');
		}

		function setLog(text) {
		  document.getElementById('action-log').textContent = sanitizeLogText(text || 'No output returned.');
		}

		function updateProgress(status) {
			const label = document.getElementById('progress-label');
			const count = document.getElementById('progress-count');
			const bar = document.getElementById('progress-bar');
			const meta = document.getElementById('progress-meta');
			if (!label || !count || !bar || !meta) {
				return;
			}
			const progress = status && status.progress ? status.progress : null;
			if (!status || !status.running) {
				if (progress && progress.total > 0) {
					label.textContent = progress.label || 'Completed';
					count.textContent = progress.current + '/' + progress.total;
					bar.value = progress.percent || 100;
					meta.textContent = status.error ? 'Failed' : 'Completed';
				} else {
					label.textContent = 'No active operation';
					count.textContent = '0/0';
					bar.value = 0;
					meta.textContent = 'Idle';
				}
				return;
			}
			if (progress) {
				label.textContent = progress.label || (status.action || 'processing');
				count.textContent = progress.current + '/' + progress.total;
				bar.value = progress.percent || 0;
				meta.textContent = 'Running ' + (status.action || 'operation') + '...';
				return;
			}
			label.textContent = status.action ? ('Running ' + status.action) : 'Running operation';
			count.textContent = '0/0';
			bar.value = 0;
			meta.textContent = 'Waiting for first progress update...';
		}

		function ensureLogPinnedBottom() {
			const logEl = document.getElementById('action-log');
			if (!logEl) {
				return;
			}
			logEl.scrollTop = logEl.scrollHeight;
		}

		function updateActionControls(status) {
			const stopBtn = document.getElementById('stop-action');
			const downloadBtn = document.getElementById('download-output');
			if (!stopBtn || !downloadBtn) {
				return;
			}
			const running = !!(status && status.running);
			const canDownload = !!(status && status.download_url);
			stopBtn.disabled = !running;
			downloadBtn.disabled = !canDownload;
			downloadBtn.dataset.url = canDownload ? status.download_url : '';
			downloadBtn.textContent = status && status.output_file ? ('Download ' + status.output_file) : 'Download output file';
		}

		function triggerOutputDownload(url) {
			if (!url) {
				return;
			}
			const link = document.createElement('a');
			link.href = url;
			link.style.display = 'none';
			document.body.appendChild(link);
			link.click();
			document.body.removeChild(link);
		}

		function normalizeLogToastMessage(level, message) {
			const posterNoCollections = /^poster mode:\s*no collections to process for library=(.+)$/i.exec(message);
			if (level === 'WARNING' && posterNoCollections && posterNoCollections[1]) {
				return 'No collections to process for library ' + posterNoCollections[1].trim();
			}
			return message;
		}

		function toastLevelFromLogLevel(level) {
			switch (level) {
			case 'ERROR':
				return 'danger';
			case 'WARNING':
				return 'warning';
			case 'SUCCESS':
				return 'success';
			default:
				return '';
			}
		}

		function processLogToasts(fullLogText) {
			const latest = fullLogText || '';
			let segment = latest;
			if (state.lastParsedLogText && latest.startsWith(state.lastParsedLogText)) {
				segment = latest.slice(state.lastParsedLogText.length);
			}
			state.lastParsedLogText = latest;
			if (!segment) {
				return;
			}
			const lines = segment.split('\n');
			for (let i = 0; i < lines.length; i++) {
				const raw = (lines[i] || '').trim();
				if (!raw || state.seenToastLogLines[raw]) {
					continue;
				}
				const match = /^(SUCCESS|WARNING|ERROR)\s+(.+)$/.exec(raw);
				if (!match) {
					continue;
				}
				const logLevel = match[1];
				const toastLevel = toastLevelFromLogLevel(logLevel);
				if (!toastLevel) {
					continue;
				}
				const message = normalizeLogToastMessage(logLevel, match[2]);
				state.seenToastLogLines[raw] = true;
				showToast(toastLevel, message, 5000);
			}
		}

		function openOperationLogAccordion() {
			if (typeof UIkit === 'undefined' || !UIkit.accordion) {
				return;
			}
			const root = document.querySelector('ul[uk-accordion]');
			if (!root) {
				return;
			}
			const accordion = UIkit.accordion(root);
			if (!accordion) {
				return;
			}
			accordion.toggle(0, true);
		}

		function configEditorLabel(scope) {
			switch (scope) {
			case 'type-collections':
				return 'Type collections list';
			case 'studio-collections':
				return 'Studio collections list';
			case 'admin-collections':
				return 'Admin collections list';
			case 'label-config':
				return 'Label config';
			case 'collection-config':
				return 'Collection config';
			default:
				return 'Config';
			}
		}

		function isCollectionListEditorScope(scope) {
			return scope === 'type-collections' || scope === 'studio-collections' || scope === 'admin-collections';
		}

		function isStructuredConfigScope(scope) {
			return scope === 'label-config' || scope === 'collection-config';
		}

		function reorderTabsAndActivatePosters() {
			const tabList = document.querySelector('ul[uk-tab]');
			const switcher = document.querySelector('ul.uk-switcher');
			if (!tabList || !switcher || tabList.children.length !== switcher.children.length) {
				return;
			}
			const tabs = Array.from(tabList.children);
			const panels = Array.from(switcher.children);
			const pairs = tabs.map((tab, idx) => ({ tab: tab, panel: panels[idx], label: ((tab.textContent || '').trim()).toLowerCase() }));
			const findPair = (name) => pairs.find((pair) => pair.label.includes(name));
			const configPair = findPair('config');
			const runtimePair = findPair('runtime');
			const postersPair = findPair('posters');
			const ordered = [];
			pairs.forEach((pair) => {
				if (pair !== configPair && pair !== runtimePair) {
					ordered.push(pair);
				}
			});
			if (configPair) {
				ordered.push(configPair);
			}
			if (runtimePair) {
				ordered.push(runtimePair);
			}
			ordered.forEach((pair) => {
				tabList.appendChild(pair.tab);
				switcher.appendChild(pair.panel);
			});
			if (typeof UIkit !== 'undefined' && UIkit.switcher && postersPair) {
				const activeIndex = ordered.findIndex((pair) => pair === postersPair);
				if (activeIndex >= 0) {
					const switcherApi = UIkit.switcher(switcher);
					if (switcherApi) {
						switcherApi.show(activeIndex);
					}
				}
			}
		}

		function updateGlobalSaveButtonVisibility() {
			const button = document.getElementById('save-config-global');
			const configForm = document.getElementById('config-form');
			if (!button || !configForm) {
				return;
			}
			const configPanel = configForm.closest('li');
			const visible = !!(configPanel && configPanel.classList.contains('uk-active'));
			if (visible) {
				button.classList.remove('fp-hidden');
				button.style.removeProperty('display');
				return;
			}
			button.classList.add('fp-hidden');
			button.style.display = 'none';
		}

		function applyActionRowLayout() {
			document.querySelectorAll('.fp-card .uk-flex').forEach((row) => {
				if (row.classList.contains('fp-button-group') || row.classList.contains('fp-action-row')) {
					return;
				}
				const directButtons = Array.from(row.children).filter((child) => child.classList && child.classList.contains('uk-button'));
				if (!directButtons.length) {
					return;
				}
				if (directButtons.length > 1) {
					row.classList.add('fp-button-group');
					return;
				}
				const hasInputOrSelect = Array.from(row.children).some((child) => {
					if (!(child instanceof HTMLElement)) {
						return false;
					}
					if (child.matches('input,select,textarea')) {
						return true;
					}
					return !!child.querySelector('input,select,textarea,.uk-input,.uk-select,.uk-textarea');
				});
				if (hasInputOrSelect) {
					row.classList.add('fp-action-row');
				}
			});
		}

		function normalizeStatsWords(raw) {
			const parts = (raw || '').split(/[\n,\t\r]+/).map((part) => (part || '').trim()).filter((part) => !!part);
			const seen = {};
			const out = [];
			parts.forEach((part) => {
				const key = part.toLowerCase();
				if (seen[key]) {
					return;
				}
				seen[key] = true;
				out.push(part);
			});
			return out;
		}

		function syncStatsWordsTextarea() {
			document.getElementById('stats-exclude-words').value = state.statsExcludeWordsItems.join(', ');
		}

		function renderStatsWordsChips() {
			const root = document.getElementById('stats-words-chip-list');
			if (!root) {
				return;
			}
			root.innerHTML = '';
			if (!state.statsExcludeWordsItems.length) {
				const empty = document.createElement('span');
				empty.className = 'fp-footer-note';
				empty.textContent = 'No excluded words added yet.';
				root.appendChild(empty);
				return;
			}
			state.statsExcludeWordsItems.forEach((word, index) => {
				const chip = document.createElement('div');
				chip.className = 'fp-inline-chip';
				chip.textContent = word;
				const remove = document.createElement('button');
				remove.type = 'button';
				remove.textContent = 'x';
				remove.addEventListener('click', () => {
					state.statsExcludeWordsItems.splice(index, 1);
					syncStatsWordsTextarea();
					renderStatsWordsChips();
				});
				chip.appendChild(remove);
				root.appendChild(chip);
			});
		}

		function addStatsWord(value) {
			const word = (value || '').trim();
			if (!word) {
				return;
			}
			const key = word.toLowerCase();
			if (state.statsExcludeWordsItems.some((existing) => existing.toLowerCase() === key)) {
				showToast('warning', 'Word already exists.', 2500);
				return;
			}
			state.statsExcludeWordsItems.push(word);
			syncStatsWordsTextarea();
			renderStatsWordsChips();
		}

		function parseCleanReplacementsRaw(raw) {
			const lines = (raw || '').split('\n');
			const out = [];
			lines.forEach((line) => {
				const trimmed = (line || '').trim();
				if (!trimmed) {
					return;
				}
				const parts = trimmed.split('=');
				if (parts.length < 2) {
					return;
				}
				const from = (parts.shift() || '').trim();
				const to = parts.join('=').trim();
				if (!from) {
					return;
				}
				out.push({ from: from, to: to });
			});
			return out;
		}

		function syncCleanReplacementsTextarea() {
			const lines = state.cleanReplacementItems.map((item) => item.from + ' = ' + item.to);
			document.getElementById('clean-replacements').value = lines.join('\n');
		}

		function renderCleanReplacementsList() {
			const root = document.getElementById('clean-repl-list');
			if (!root) {
				return;
			}
			root.innerHTML = '';
			if (!state.cleanReplacementItems.length) {
				const empty = document.createElement('div');
				empty.className = 'fp-config-list-empty';
				empty.textContent = 'No replacements added yet.';
				root.appendChild(empty);
				return;
			}
			state.cleanReplacementItems.forEach((item, index) => {
				const row = document.createElement('div');
				row.className = 'fp-config-list-row';
				const value = document.createElement('div');
				value.className = 'fp-config-list-value';
				value.textContent = item.from + ' => ' + item.to;
				const remove = document.createElement('button');
				remove.type = 'button';
				remove.className = 'uk-button uk-button-danger uk-button-small';
				remove.textContent = 'Remove';
				remove.addEventListener('click', () => {
					state.cleanReplacementItems.splice(index, 1);
					syncCleanReplacementsTextarea();
					renderCleanReplacementsList();
				});
				row.appendChild(value);
				row.appendChild(remove);
				root.appendChild(row);
			});
		}

		function addCleanReplacement(fromValue, toValue) {
			const from = (fromValue || '').trim();
			const to = (toValue || '').trim();
			if (!from) {
				showToast('warning', 'Find text is required.', 2500);
				return;
			}
			const duplicate = state.cleanReplacementItems.some((item) => item.from.toLowerCase() === from.toLowerCase());
			if (duplicate) {
				showToast('warning', 'Replacement for that key already exists.', 2500);
				return;
			}
			state.cleanReplacementItems.push({ from: from, to: to });
			syncCleanReplacementsTextarea();
			renderCleanReplacementsList();
		}

		function structuredRowsFromLabelConfig(obj) {
			const rows = [];
			const list = obj && obj.label && Array.isArray(obj.label.lookup) ? obj.label.lookup : [];
			list.forEach((item) => {
				rows.push({
					title_contains: item.title_contains || '',
					title_contains_any: Array.isArray(item.title_contains_any) ? item.title_contains_any.join(', ') : '',
					find: item.find || '',
					labels: Array.isArray(item.labels) ? item.labels.join(', ') : '',
					categories: Array.isArray(item.categories) ? item.categories.join(', ') : '',
					update_category: !!item.update_category,
					only_category: !!item.only_category
				});
			});
			return rows;
		}

		function structuredRowsFromCollectionConfig(obj) {
			const rows = [];
			const list = obj && obj.collection && Array.isArray(obj.collection.lookup) ? obj.collection.lookup : [];
			list.forEach((item) => {
				rows.push({
					title: item.title || '',
					smart: !!item.smart,
					content: item.content || ''
				});
			});
			return rows;
		}

		function parseCsvList(raw) {
			return (raw || '').split(',').map((x) => x.trim()).filter((x) => !!x);
		}

		function renderStructuredConfigEditor() {
			const root = document.getElementById('config-editor-structured');
			if (!root) {
				return;
			}
			root.innerHTML = '';
			if (!isStructuredConfigScope(state.configEditorScope)) {
				return;
			}
			if (!state.configEditorStructuredReady) {
				const warning = document.createElement('div');
				warning.className = 'fp-config-list-empty';
				warning.textContent = 'Unable to parse structured view. Use Raw text mode.';
				root.appendChild(warning);
				return;
			}
			if (state.configEditorScope === 'label-config') {
				renderLabelStructuredEditor(root);
				return;
			}
			renderCollectionStructuredEditor(root);
		}

		function renderLabelStructuredEditor(root) {
			const list = state.configEditorStructuredContent;
			const rows = structuredRowsFromLabelConfig(list);
			const addButton = document.createElement('button');
			addButton.type = 'button';
			addButton.className = 'uk-button uk-button-secondary uk-button-small';
			addButton.textContent = 'Add lookup';
			addButton.addEventListener('click', () => {
				if (!list.label) {
					list.label = {};
				}
				if (!Array.isArray(list.label.lookup)) {
					list.label.lookup = [];
				}
				list.label.lookup.push({
					title_contains: '',
					title_contains_any: [],
					find: '',
					labels: [],
					categories: [],
					update_category: false,
					only_category: false
				});
				renderStructuredConfigEditor();
			});
			root.appendChild(addButton);
			if (!rows.length) {
				const empty = document.createElement('div');
				empty.className = 'fp-config-list-empty';
				empty.textContent = 'No lookups configured yet.';
				root.appendChild(empty);
				return;
			}
			list.label.lookup.forEach((item, index) => {
				const card = document.createElement('div');
				card.className = 'fp-lookup-card';
				card.innerHTML = '<div class="uk-grid-small" uk-grid>' +
					'<div class="uk-width-1-2"><label class="uk-form-label">Title contains</label><input class="uk-input" data-k="title_contains" type="text"></div>' +
					'<div class="uk-width-1-2"><label class="uk-form-label">Find</label><input class="uk-input" data-k="find" type="text"></div>' +
					'<div class="uk-width-1-1"><label class="uk-form-label">Title contains any (comma)</label><input class="uk-input" data-k="title_contains_any" type="text"></div>' +
					'<div class="uk-width-1-2"><label class="uk-form-label">Labels (comma)</label><input class="uk-input" data-k="labels" type="text"></div>' +
					'<div class="uk-width-1-2"><label class="uk-form-label">Categories (comma)</label><input class="uk-input" data-k="categories" type="text"></div>' +
					'<div class="uk-width-1-1 uk-flex uk-gap-small uk-flex-wrap uk-flex-middle"><label><input class="uk-checkbox" data-k="update_category" type="checkbox"> Update category</label><label><input class="uk-checkbox" data-k="only_category" type="checkbox"> Only category</label><button type="button" class="uk-button uk-button-danger uk-button-small" data-k="remove">Remove</button></div>' +
					'</div>';
				card.querySelector('[data-k="title_contains"]').value = item.title_contains || '';
				card.querySelector('[data-k="find"]').value = item.find || '';
				card.querySelector('[data-k="title_contains_any"]').value = Array.isArray(item.title_contains_any) ? item.title_contains_any.join(', ') : '';
				card.querySelector('[data-k="labels"]').value = Array.isArray(item.labels) ? item.labels.join(', ') : '';
				card.querySelector('[data-k="categories"]').value = Array.isArray(item.categories) ? item.categories.join(', ') : '';
				card.querySelector('[data-k="update_category"]').checked = !!item.update_category;
				card.querySelector('[data-k="only_category"]').checked = !!item.only_category;
				card.querySelectorAll('input').forEach((input) => {
					input.addEventListener('input', () => {
						item.title_contains = card.querySelector('[data-k="title_contains"]').value;
						item.find = card.querySelector('[data-k="find"]').value;
						item.title_contains_any = parseCsvList(card.querySelector('[data-k="title_contains_any"]').value);
						item.labels = parseCsvList(card.querySelector('[data-k="labels"]').value);
						item.categories = parseCsvList(card.querySelector('[data-k="categories"]').value);
						item.update_category = card.querySelector('[data-k="update_category"]').checked;
						item.only_category = card.querySelector('[data-k="only_category"]').checked;
					});
				});
				card.querySelector('[data-k="remove"]').addEventListener('click', () => {
					list.label.lookup.splice(index, 1);
					renderStructuredConfigEditor();
				});
				root.appendChild(card);
			});
		}

		function renderCollectionStructuredEditor(root) {
			const list = state.configEditorStructuredContent;
			const rows = structuredRowsFromCollectionConfig(list);
			const addButton = document.createElement('button');
			addButton.type = 'button';
			addButton.className = 'uk-button uk-button-secondary uk-button-small';
			addButton.textContent = 'Add collection rule';
			addButton.addEventListener('click', () => {
				if (!list.collection) {
					list.collection = {};
				}
				if (!Array.isArray(list.collection.lookup)) {
					list.collection.lookup = [];
				}
				list.collection.lookup.push({ title: '', smart: true, content: '' });
				renderStructuredConfigEditor();
			});
			root.appendChild(addButton);
			if (!rows.length) {
				const empty = document.createElement('div');
				empty.className = 'fp-config-list-empty';
				empty.textContent = 'No collection lookup rules configured yet.';
				root.appendChild(empty);
				return;
			}
			list.collection.lookup.forEach((item, index) => {
				const card = document.createElement('div');
				card.className = 'fp-lookup-card';
				card.innerHTML = '<div class="uk-grid-small" uk-grid>' +
					'<div class="uk-width-2-3"><label class="uk-form-label">Title</label><input class="uk-input" data-k="title" type="text"></div>' +
					'<div class="uk-width-1-3 uk-flex uk-flex-middle"><label><input class="uk-checkbox" data-k="smart" type="checkbox"> Smart collection</label></div>' +
					'<div class="uk-width-1-1"><label class="uk-form-label">Content query</label><textarea class="uk-textarea" rows="3" data-k="content"></textarea></div>' +
					'<div class="uk-width-1-1"><button type="button" class="uk-button uk-button-danger uk-button-small" data-k="remove">Remove</button></div>' +
					'</div>';
				card.querySelector('[data-k="title"]').value = item.title || '';
				card.querySelector('[data-k="smart"]').checked = !!item.smart;
				card.querySelector('[data-k="content"]').value = item.content || '';
				card.querySelectorAll('input,textarea').forEach((input) => {
					input.addEventListener('input', () => {
						item.title = card.querySelector('[data-k="title"]').value;
						item.smart = card.querySelector('[data-k="smart"]').checked;
						item.content = card.querySelector('[data-k="content"]').value;
					});
				});
				card.querySelector('[data-k="remove"]').addEventListener('click', () => {
					list.collection.lookup.splice(index, 1);
					renderStructuredConfigEditor();
				});
				root.appendChild(card);
			});
		}

		function syncStructuredRawTextFromObject() {
			if (!isStructuredConfigScope(state.configEditorScope)) {
				return;
			}
			const source = state.configEditorStructuredContent || {};
			const lines = [];
			if (state.configEditorScope === 'label-config') {
				const lookups = source.label && Array.isArray(source.label.lookup) ? source.label.lookup : [];
				lookups.forEach((entry, idx) => {
					if (idx > 0) {
						lines.push('');
					}
					lines.push('[[label.lookup]]');
					if (entry.title_contains) {
						lines.push('title_contains = ' + JSON.stringify(entry.title_contains));
					}
					if (Array.isArray(entry.title_contains_any) && entry.title_contains_any.length) {
						lines.push('title_contains_any = [' + entry.title_contains_any.map((item) => JSON.stringify(item)).join(', ') + ']');
					}
					if (entry.find) {
						lines.push('find = ' + JSON.stringify(entry.find));
					}
					if (Array.isArray(entry.labels) && entry.labels.length) {
						lines.push('labels = [' + entry.labels.map((item) => JSON.stringify(item)).join(', ') + ']');
					}
					if (Array.isArray(entry.categories) && entry.categories.length) {
						lines.push('categories = [' + entry.categories.map((item) => JSON.stringify(item)).join(', ') + ']');
					}
					lines.push('update_category = ' + (entry.update_category ? 'true' : 'false'));
					lines.push('only_category = ' + (entry.only_category ? 'true' : 'false'));
				});
			} else {
				if (source.base_uri) {
					lines.push('base_uri = ' + JSON.stringify(source.base_uri));
					lines.push('');
				}
				const lookups = source.collection && Array.isArray(source.collection.lookup) ? source.collection.lookup : [];
				lookups.forEach((entry, idx) => {
					if (idx > 0) {
						lines.push('');
					}
					lines.push('[[collection.lookup]]');
					lines.push('title = ' + JSON.stringify(entry.title || ''));
					lines.push('smart = ' + (entry.smart ? 'true' : 'false'));
					lines.push('content = ' + JSON.stringify(entry.content || ''));
				});
			}
			state.configEditorRawContent = lines.join('\n');
			document.getElementById('config-editor-text').value = state.configEditorRawContent;
		}

		function applyConfigEditorMode(mode) {
			const nextMode = mode === 'raw' ? 'raw' : 'structured';
			state.configEditorMode = nextMode;
			const listMode = document.getElementById('config-editor-list-mode');
			const structured = document.getElementById('config-editor-structured');
			const textarea = document.getElementById('config-editor-text');
			const modeSwitch = document.getElementById('config-editor-mode-switch');
			const structuredBtn = document.getElementById('config-editor-mode-structured');
			const rawBtn = document.getElementById('config-editor-mode-raw');
			const listScope = isCollectionListEditorScope(state.configEditorScope);
			const structuredScope = isStructuredConfigScope(state.configEditorScope);
			if (listScope) {
				modeSwitch.hidden = true;
				listMode.hidden = false;
				structured.hidden = true;
				textarea.hidden = true;
				return;
			}
			if (structuredScope) {
				modeSwitch.hidden = false;
				listMode.hidden = true;
				structured.hidden = nextMode !== 'structured';
				textarea.hidden = nextMode !== 'raw';
				structuredBtn.className = 'uk-button uk-button-small ' + (nextMode === 'structured' ? 'uk-button-primary' : 'uk-button-default');
				rawBtn.className = 'uk-button uk-button-small ' + (nextMode === 'raw' ? 'uk-button-primary' : 'uk-button-default');
				if (nextMode === 'structured') {
					renderStructuredConfigEditor();
				} else {
					document.getElementById('config-editor-text').value = state.configEditorRawContent || '';
				}
				return;
			}
			modeSwitch.hidden = true;
			listMode.hidden = true;
			structured.hidden = true;
			textarea.hidden = false;
		}

		function parseCollectionListContent(text) {
			return (text || '')
				.split('\n')
				.map((line) => (line || '').trim())
				.filter((line) => !!line);
		}

		function renderConfigEditorList() {
			const listRoot = document.getElementById('config-editor-list');
			if (!listRoot) {
				return;
			}
			listRoot.innerHTML = '';
			if (!state.configEditorItems.length) {
				const empty = document.createElement('div');
				empty.className = 'fp-config-list-empty';
				empty.textContent = 'No entries yet. Add one above.';
				listRoot.appendChild(empty);
				return;
			}
			state.configEditorItems.forEach((item, index) => {
				const row = document.createElement('div');
				row.className = 'fp-config-list-row';
				const value = document.createElement('div');
				value.className = 'fp-config-list-value';
				value.textContent = item;
				const remove = document.createElement('button');
				remove.type = 'button';
				remove.className = 'uk-button uk-button-danger uk-button-small';
				remove.textContent = 'Remove';
				remove.addEventListener('click', () => {
					state.configEditorItems.splice(index, 1);
					renderConfigEditorList();
				});
				row.appendChild(value);
				row.appendChild(remove);
				listRoot.appendChild(row);
			});
		}

		function addConfigEditorItem(rawValue) {
			const value = (rawValue || '').trim();
			if (!value) {
				return;
			}
			const exists = state.configEditorItems.some((item) => item.toLowerCase() === value.toLowerCase());
			if (exists) {
				showToast('warning', 'Entry already exists.', 2500);
				return;
			}
			state.configEditorItems.push(value);
			renderConfigEditorList();
		}

		function applyConfigEditorModeForScope(scope, content) {
			const listMode = document.getElementById('config-editor-list-mode');
			const structured = document.getElementById('config-editor-structured');
			const modeSwitch = document.getElementById('config-editor-mode-switch');
			const textarea = document.getElementById('config-editor-text');
			const input = document.getElementById('config-editor-item-input');
			if (!listMode || !structured || !modeSwitch || !textarea || !input) {
				return;
			}
			state.configEditorRawContent = content || '';
			state.configEditorStructuredReady = false;
			state.configEditorStructuredContent = '';
			if (isCollectionListEditorScope(scope)) {
				state.configEditorItems = parseCollectionListContent(content);
				modeSwitch.hidden = true;
				listMode.hidden = false;
				structured.hidden = true;
				textarea.hidden = true;
				textarea.value = '';
				input.value = '';
				renderConfigEditorList();
				return;
			}
			if (isStructuredConfigScope(scope)) {
				try {
					state.configEditorStructuredContent = JSON.parse(content || '{}');
					state.configEditorStructuredReady = true;
				} catch (error) {
					state.configEditorStructuredContent = {};
					state.configEditorStructuredReady = false;
				}
				syncStructuredRawTextFromObject();
				applyConfigEditorMode('structured');
				return;
			}
			state.configEditorItems = [];
			modeSwitch.hidden = true;
			listMode.hidden = true;
			structured.hidden = true;
			textarea.hidden = false;
			textarea.value = content || '';
		}

		function buildConfigEditorContentForSave() {
			if (isCollectionListEditorScope(state.configEditorScope)) {
				return state.configEditorItems.join('\n');
			}
			if (isStructuredConfigScope(state.configEditorScope)) {
				if (state.configEditorMode === 'raw') {
					return document.getElementById('config-editor-text').value;
				}
				syncStructuredRawTextFromObject();
				return state.configEditorRawContent;
			}
			return document.getElementById('config-editor-text').value;
		}

		function syncTemplateSelection(selectEl) {
			if (!selectEl) {
				return;
			}
			const manager = selectEl.closest('.fp-template-manager');
			if (!manager) {
				return;
			}
			const targetId = manager.dataset.targetInput;
			const hidden = document.getElementById(targetId);
			if (!hidden) {
				return;
			}
			hidden.value = selectEl.value || '';
		}

		function ensureTemplateSelectOption(selectEl, value) {
			if (!selectEl || !value) {
				return;
			}
			const existing = Array.from(selectEl.options).find((option) => option.value === value);
			if (!existing) {
				const option = document.createElement('option');
				option.value = value;
				option.textContent = value;
				selectEl.appendChild(option);
			}
		}

		function populateTemplateSelects() {
			const items = Array.isArray(state.templateFiles) ? state.templateFiles : [];
			document.querySelectorAll('.fp-template-manager').forEach((manager) => {
				const selectEl = manager.querySelector('.fp-template-select');
				if (!selectEl) {
					return;
				}
				const targetId = manager.dataset.targetInput;
				const hidden = document.getElementById(targetId);
				const currentValue = hidden ? (hidden.value || '') : '';
				selectEl.innerHTML = '';
				items.forEach((item) => {
					const option = document.createElement('option');
					option.value = item.value || item.name || '';
					option.textContent = item.name || item.value || 'template image';
					selectEl.appendChild(option);
				});
				if (!items.length) {
					const option = document.createElement('option');
					option.value = currentValue;
					option.textContent = currentValue || 'No uploaded template images found';
					selectEl.appendChild(option);
				}
				if (currentValue) {
					ensureTemplateSelectOption(selectEl, currentValue);
					selectEl.value = currentValue;
				}
				syncTemplateSelection(selectEl);
			});
		}

		async function refreshTemplateFiles() {
			const response = await fetch('/api/files/list?scope=template-images');
			const payload = await response.json();
			if (!response.ok) {
				throw new Error(payload.error || 'Failed to load template images');
			}
			state.templateFiles = payload.files || [];
			populateTemplateSelects();
		}

		async function uploadTemplateImageFromManager(manager) {
			if (!manager) {
				throw new Error('Template upload control is unavailable.');
			}
			const uploadInput = manager.querySelector('.fp-template-upload');
			if (!uploadInput || !uploadInput.files || !uploadInput.files.length) {
				throw new Error('Choose an image file first.');
			}
			const file = uploadInput.files[0];
			const formData = new FormData();
			formData.append('file', file);
			const response = await fetch('/api/files/upload?scope=template-images', {
				method: 'POST',
				body: formData
			});
			const payload = await response.json();
			if (!response.ok || !payload.ok) {
				throw new Error(payload.error || 'Template image upload failed');
			}
			state.templateFiles = payload.files || [];
			populateTemplateSelects();
			const selectEl = manager.querySelector('.fp-template-select');
			if (selectEl && payload.value) {
				ensureTemplateSelectOption(selectEl, payload.value);
				selectEl.value = payload.value;
				syncTemplateSelection(selectEl);
			}
			uploadInput.value = '';
			return payload;
		}

		async function openConfigEditor(scope) {
			const response = await fetch('/api/config/content?scope=' + encodeURIComponent(scope));
			const payload = await response.json();
			if (!response.ok) {
				throw new Error(payload.error || 'Failed to load config content');
			}
			state.configEditorScope = scope;
			state.configEditorTitle = configEditorLabel(scope);
			document.getElementById('config-editor-title').textContent = state.configEditorTitle;
			document.getElementById('config-editor-path').textContent = payload.path || '';
			applyConfigEditorModeForScope(scope, payload.content || '');
			document.getElementById('config-editor-hint').textContent = 'Edit and save.';
			if (typeof UIkit !== 'undefined' && UIkit.modal) {
				UIkit.modal('#config-editor-modal').show();
			}
		}

		async function saveConfigEditorContent() {
			if (!state.configEditorScope) {
				throw new Error('No config editor scope is selected.');
			}
			const body = {
				scope: state.configEditorScope,
				content: buildConfigEditorContentForSave()
			};
			const result = await postJSON('/api/config/content?scope=' + encodeURIComponent(state.configEditorScope), body);
			document.getElementById('config-editor-hint').textContent = result.message || 'Saved.';
			showToast('success', result.message || (state.configEditorTitle + ' saved.'), 3500);
			await refreshState();
		}

		function setImportFileOptions(files, preferredValue) {
			const select = document.getElementById('import-file');
			if (!select) {
				return;
			}
			const items = Array.isArray(files) ? files : [];
			const previous = preferredValue || select.value || '';
			select.innerHTML = '';
			if (!items.length) {
				const option = document.createElement('option');
				option.value = previous;
				option.textContent = previous || 'No import files found';
				select.appendChild(option);
				return;
			}
			items.forEach((item) => {
				const option = document.createElement('option');
				option.value = item.value || item.name || '';
				option.textContent = item.name || item.value || 'import file';
				select.appendChild(option);
			});
			if (previous) {
				const match = Array.from(select.options).find((option) => option.value === previous || option.textContent === previous);
				if (match) {
					select.value = match.value;
					return;
				}
				const custom = document.createElement('option');
				custom.value = previous;
				custom.textContent = previous;
				select.appendChild(custom);
				select.value = previous;
				return;
			}
			select.value = select.options[0].value;
		}

		async function refreshImportFiles(preferredValue) {
			const response = await fetch('/api/files/list?scope=collections-import');
			const payload = await response.json();
			if (!response.ok) {
				throw new Error(payload.error || 'Failed to load import files');
			}
			setImportFileOptions(payload.files || [], preferredValue || payload.default || '');
		}

		async function uploadImportFile() {
			const input = document.getElementById('import-file-upload');
			if (!input || !input.files || !input.files.length) {
				throw new Error('Choose a JSON import file first.');
			}
			const file = input.files[0];
			const formData = new FormData();
			formData.append('file', file);
			const response = await fetch('/api/files/upload?scope=collections-import', {
				method: 'POST',
				body: formData
			});
			const payload = await response.json();
			if (!response.ok || !payload.ok) {
				throw new Error(payload.error || 'Failed to upload import file');
			}
			setImportFileOptions(payload.files || [], payload.value || payload.file || file.name);
			input.value = '';
			return payload;
		}

		async function pollActionStatus() {
			if (state.actionPollBusy) {
				return;
			}
			state.actionPollBusy = true;
			try {
				const response = await fetch('/api/action/status');
				if (!response.ok) {
					return;
				}
				const payload = await response.json();
				const currentHash = JSON.stringify({
					running: payload.running,
					action: payload.action,
					canceled: payload.canceled,
					stop_asked: payload.stop_asked,
					error: payload.error,
					message: payload.message,
					output_file: payload.output_file,
					download_url: payload.download_url,
					logs: payload.logs,
					progress: payload.progress
				});
				if (currentHash !== state.lastActionStatusHash) {
					state.lastActionStatusHash = currentHash;
					updateProgress(payload);
					updateActionControls(payload);
					if (payload.logs) {
						setLog(payload.logs);
						processLogToasts(payload.logs);
						ensureLogPinnedBottom();
					}
					if (!payload.running && payload.download_url) {
						const token = (payload.completed_at || '') + '|' + payload.download_url;
						if (state.lastDownloadToken !== token) {
							state.lastDownloadToken = token;
							triggerOutputDownload(payload.download_url);
						}
					}
				}
			} catch (error) {
				// Keep polling; transient issues should not break UX.
			} finally {
				state.actionPollBusy = false;
			}
		}

		function startActionPolling() {
			if (state.actionPollTimer) {
				return;
			}
			pollActionStatus();
			state.actionPollTimer = setInterval(pollActionStatus, state.actionPollFast);
		}

		function stopActionPolling() {
			if (!state.actionPollTimer) {
				return;
			}
			clearInterval(state.actionPollTimer);
			state.actionPollTimer = null;
			setTimeout(pollActionStatus, 0);
		}

    function updateStatusChip(valid, message) {
      const chip = document.getElementById('config-status-chip');
      chip.textContent = message;
      chip.style.background = valid ? 'rgba(45,139,87,0.12)' : 'rgba(157,47,47,0.12)';
      chip.style.color = valid ? 'var(--fp-ok)' : 'var(--fp-error)';
    }

		function clearPlexTestButtonState() {
			const button = document.getElementById('test-plex-connection');
			const icon = document.getElementById('test-plex-connection-icon');
			if (!button || !icon) {
				return;
			}
			button.classList.remove('is-loading', 'is-success', 'is-error');
			button.disabled = false;
			icon.textContent = '';
			if (state.plexTestResetTimer) {
				clearTimeout(state.plexTestResetTimer);
				state.plexTestResetTimer = null;
			}
		}

		function setPlexTestButtonState(nextState) {
			const button = document.getElementById('test-plex-connection');
			const icon = document.getElementById('test-plex-connection-icon');
			if (!button || !icon) {
				return;
			}
			clearPlexTestButtonState();
			if (nextState === 'loading') {
				button.classList.add('is-loading');
				button.disabled = true;
				icon.textContent = '...';
				return;
			}
			if (nextState === 'success') {
				button.classList.add('is-success');
				icon.textContent = '✓';
				state.plexTestResetTimer = setTimeout(clearPlexTestButtonState, 4500);
				return;
			}
			if (nextState === 'error') {
				button.classList.add('is-error');
				icon.textContent = '✕';
				state.plexTestResetTimer = setTimeout(clearPlexTestButtonState, 4500);
			}
		}

    function makeOption(section) {
      const option = document.createElement('option');
      option.value = section.key;
	option.textContent = section.title + ' (' + section.type + ')';
      return option;
    }

    function fillSectionSelect(selectId, allowPlaceholder) {
      const select = document.getElementById(selectId);
      if (!select) return;
      const previous = select.multiple ? Array.from(select.selectedOptions).map((option) => option.value) : select.value;
      select.innerHTML = '';
      if (allowPlaceholder) {
        const placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.textContent = state.sections.length ? 'Choose a library' : 'No libraries loaded';
        select.appendChild(placeholder);
      }
      state.sections.forEach((section) => select.appendChild(makeOption(section)));
      if (select.multiple) {
        previous.forEach((value) => {
          const match = Array.from(select.options).find((option) => option.value === value);
          if (match) match.selected = true;
        });
      } else if (previous) {
        select.value = previous;
      }
      if (!select.multiple && !select.value && state.sections.length && !allowPlaceholder) {
        select.value = state.sections[0].key;
      }
    }

    async function loadCollectionsForPathClean() {
      const sectionKey = document.getElementById('path-clean-section').value;
      const collectionSelect = document.getElementById('path-clean-collection');
      collectionSelect.innerHTML = '';
      if (!sectionKey) {
        return;
      }
      try {
		const response = await fetch('/api/sections/' + encodeURIComponent(sectionKey) + '/collections');
        const payload = await response.json();
        if (!response.ok) throw new Error(payload.error || 'Failed to load collections');
        payload.collections.forEach((collection) => {
          const option = document.createElement('option');
          option.value = collection.rating_key;
          option.textContent = collection.title;
          collectionSelect.appendChild(option);
        });
      } catch (error) {
        showBanner('error', error.message);
      }
    }

    async function refreshState() {
      const response = await fetch('/api/state');
      const payload = await response.json();
      if (!response.ok) {
        showBanner('error', payload.error || 'Failed to load state');
        return;
      }
      state.sections = payload.sections || [];
      state.backups = payload.backups || [];
      state.defaults = payload.defaults || {};

      document.getElementById('base-url').value = payload.plex.base_url || '';
      document.getElementById('token').value = payload.plex.token || '';
      document.getElementById('retries').value = payload.plex.retries || 1;
      document.getElementById('workers').value = payload.plex.workers || 1;
      document.getElementById('retry-base-ms').value = payload.plex.retry_base_ms || 500;
      document.getElementById('retry-max-ms').value = payload.plex.retry_max_ms || 30000;
	document.getElementById('output-dir-input').value = payload.general.output_dir || '';
	document.getElementById('log-file-input').value = payload.general.log_file || '';
	document.getElementById('translate-endpoint').value = payload.general.translate_endpoint || '';
	document.getElementById('translate-api-key').value = payload.general.translate_api_key || '';
	document.getElementById('translate-rate-limit').value = payload.general.translate_rate_limit_per_minute || 10;
	document.getElementById('template-image').value = payload.general.template_image || '';
	document.getElementById('type-template-image').value = payload.general.type_template_image || '';
	document.getElementById('studio-template-image').value = payload.general.studio_template_image || '';
	document.getElementById('admin-template-image').value = payload.general.admin_template_image || '';
	document.getElementById('type-collections-file').value = payload.general.type_collections_file || '';
	document.getElementById('studio-collections-file').value = payload.general.studio_collections_file || '';
	document.getElementById('admin-collections-file').value = payload.general.admin_collections_file || '';
	document.getElementById('plex-config-file').value = payload.general.plex_config || '';
	document.getElementById('label-config-file').value = payload.general.label_config || '';
	document.getElementById('collection-config-file').value = payload.general.collection_config || '';
	document.getElementById('translate-to-english').checked = !!payload.general.translate_to_english;
	document.getElementById('clean-replacements').value = payload.general.clean_replacements || '';
	document.getElementById('stats-exclude-words').value = payload.general.stats_exclude_words || '';
		state.cleanReplacementItems = parseCleanReplacementsRaw(payload.general.clean_replacements || '');
		renderCleanReplacementsList();
		state.statsExcludeWordsItems = normalizeStatsWords(payload.general.stats_exclude_words || '');
		renderStatsWordsChips();
	document.getElementById('backup-retention-days').value = payload.general.backup_retention_days || 0;
	document.getElementById('font-file').value = payload.general.font_file || '';
	document.getElementById('font-size').value = payload.general.font_size || 0;
	document.getElementById('font-color').value = payload.general.font_color || '';
	document.getElementById('font-shadow-color').value = payload.general.font_shadow_color || '';
	document.getElementById('font-shadow-offset-x').value = payload.general.font_shadow_offset_x || 0;
	document.getElementById('font-shadow-offset-y').value = payload.general.font_shadow_offset_y || 0;
	document.getElementById('font-glow-color').value = payload.general.font_glow_color || '';
	document.getElementById('font-glow-radius').value = payload.general.font_glow_radius || 0;
	document.getElementById('font-glow-alpha').value = payload.general.font_glow_alpha || 0;
	document.getElementById('font-y-offset').value = payload.general.font_y_offset || 0;
      document.getElementById('config-path').textContent = payload.config_path;
      document.getElementById('output-dir').textContent = payload.output_dir;
      document.getElementById('log-file').textContent = payload.log_file;
      document.getElementById('export-file').value = payload.defaults.export_path || '';
      document.getElementById('import-file').value = payload.defaults.import_path || '';
		try {
			await refreshImportFiles(payload.defaults.import_path || '');
		} catch (error) {
			showToast('warning', error.message, 4500);
		}
		try {
			await refreshTemplateFiles();
		} catch (error) {
			showToast('warning', error.message, 4500);
		}

      sectionSelectIds.forEach((id) => fillSectionSelect(id, false));
      fillSectionSelect('poster-sections', false);
      await loadCollectionsForPathClean();

      const backupList = document.getElementById('backup-list');
      if (!state.backups.length) {
        backupList.textContent = 'No backups found.';
      } else {
		backupList.innerHTML = state.backups.map((backup) => '<span class="fp-chip">' + backup.name + '</span>').join('');
      }

      if (payload.config_valid) {
		const sectionsSummary = payload.sections_error ? 'Config valid. Plex fetch issue: ' + payload.sections_error : 'Config valid. Libraries loaded: ' + state.sections.length;
		if (payload.sections_error) {
			showBanner('warn', sectionsSummary);
		} else {
			hideBanner();
			if (!state.runtimeStatusToastShown) {
				showToast('success', sectionsSummary, 4000);
				state.runtimeStatusToastShown = true;
			}
		}
        updateStatusChip(true, sectionsSummary);
      } else {
        const errorMessage = payload.config_error || 'Config validation failed.';
        showBanner('warn', errorMessage);
        updateStatusChip(false, errorMessage);
      }
		updateActionControls(null);
    }

    async function postJSON(url, body) {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {})
      });
      const payload = await response.json();
      if (!response.ok || !payload.ok) {
        throw new Error(payload.error || 'Request failed');
      }
      return payload;
    }

		function buildConfigPayload() {
			return {
				base_url: document.getElementById('base-url').value,
				token: document.getElementById('token').value,
				retries: Number(document.getElementById('retries').value),
				workers: Number(document.getElementById('workers').value),
				retry_base_ms: Number(document.getElementById('retry-base-ms').value),
				retry_max_ms: Number(document.getElementById('retry-max-ms').value),
				output_dir: document.getElementById('output-dir-input').value,
				log_file: document.getElementById('log-file-input').value,
				template_image: document.getElementById('template-image').value,
				type_template_image: document.getElementById('type-template-image').value,
				studio_template_image: document.getElementById('studio-template-image').value,
				admin_template_image: document.getElementById('admin-template-image').value,
				type_collections_file: document.getElementById('type-collections-file').value,
				studio_collections_file: document.getElementById('studio-collections-file').value,
				admin_collections_file: document.getElementById('admin-collections-file').value,
				plex_config: document.getElementById('plex-config-file').value,
				label_config: document.getElementById('label-config-file').value,
				collection_config: document.getElementById('collection-config-file').value,
				translate_to_english: document.getElementById('translate-to-english').checked,
				translate_endpoint: document.getElementById('translate-endpoint').value,
				translate_api_key: document.getElementById('translate-api-key').value,
				translate_rate_limit_per_minute: Number(document.getElementById('translate-rate-limit').value),
				clean_replacements: document.getElementById('clean-replacements').value,
				stats_exclude_words: document.getElementById('stats-exclude-words').value,
				backup_retention_days: Number(document.getElementById('backup-retention-days').value),
				font_file: document.getElementById('font-file').value,
				font_size: Number(document.getElementById('font-size').value),
				font_color: document.getElementById('font-color').value,
				font_shadow_color: document.getElementById('font-shadow-color').value,
				font_shadow_offset_x: Number(document.getElementById('font-shadow-offset-x').value),
				font_shadow_offset_y: Number(document.getElementById('font-shadow-offset-y').value),
				font_glow_color: document.getElementById('font-glow-color').value,
				font_glow_radius: Number(document.getElementById('font-glow-radius').value),
				font_glow_alpha: Number(document.getElementById('font-glow-alpha').value),
				font_y_offset: Number(document.getElementById('font-y-offset').value)
			};
		}

		async function saveConfigPayload(options) {
			const config = options || {};
			const payload = buildConfigPayload();
			const result = await postJSON('/api/config', payload);
			if (!config.quiet) {
				setLog(result.logs || result.message || 'Configuration updated.');
				showBanner('ok', result.message || 'Configuration saved.');
				showToast('success', config.successMessage || result.message || 'Configuration saved.', 3500);
			}
			await refreshState();
			return result;
		}

		function buildTemplatePreviewPayload() {
			return {
				template_kind: document.getElementById('preview-template-kind').value,
				sample_text: document.getElementById('preview-sample-text').value,
				template_image: document.getElementById('template-image').value,
				type_template_image: document.getElementById('type-template-image').value,
				studio_template_image: document.getElementById('studio-template-image').value,
				admin_template_image: document.getElementById('admin-template-image').value,
				font_file: document.getElementById('font-file').value,
				font_size: Number(document.getElementById('font-size').value),
				font_color: document.getElementById('font-color').value,
				font_shadow_color: document.getElementById('font-shadow-color').value,
				font_shadow_offset_x: Number(document.getElementById('font-shadow-offset-x').value),
				font_shadow_offset_y: Number(document.getElementById('font-shadow-offset-y').value),
				font_glow_color: document.getElementById('font-glow-color').value,
				font_glow_radius: Number(document.getElementById('font-glow-radius').value),
				font_glow_alpha: Number(document.getElementById('font-glow-alpha').value),
				font_y_offset: Number(document.getElementById('font-y-offset').value)
			};
		}

		async function requestTemplatePreview() {
			const button = document.getElementById('generate-template-preview');
			const image = document.getElementById('template-preview-image');
			const empty = document.getElementById('template-preview-empty');
			const meta = document.getElementById('template-preview-meta');
			button.disabled = true;
			meta.textContent = 'Generating preview...';
			try {
				const response = await fetch('/api/template/preview', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(buildTemplatePreviewPayload())
				});
				const payload = await response.json();
				if (!response.ok || !payload.ok) {
					throw new Error(payload.error || 'Failed to generate preview');
				}
				image.src = payload.image_data_url;
				image.style.display = 'block';
				empty.style.display = 'none';
				meta.textContent = 'Preview generated from ' + (payload.template_kind || 'template') + ' (' + payload.width + 'x' + payload.height + ') using ' + (payload.template_path || 'selected path');
				showToast('success', 'Template preview generated.', 2500);
			} catch (error) {
				meta.textContent = error.message;
				showToast('danger', error.message, 5000);
			} finally {
				button.disabled = false;
			}
		}

    async function runAction(action, payload) {
	openOperationLogAccordion();
	state.lastParsedLogText = '';
	state.seenToastLogLines = {};
	setLog('Running ' + action + '...');
	state.lastActionStatusHash = '';
	updateProgress({ running: true, action: action });
	startActionPolling();
      try {
		const result = await postJSON('/api/action/' + action, payload);
        setLog(result.logs || result.message || 'Action completed.');
		showBanner('ok', result.message || (action + ' completed.'));
        await refreshState();
      } catch (error) {
        setLog(error.message);
        showBanner('error', error.message);
		} finally {
			stopActionPolling();
      }
    }

    document.getElementById('refresh-state').addEventListener('click', refreshState);
    document.getElementById('path-clean-section').addEventListener('change', loadCollectionsForPathClean);

		document.querySelectorAll('[data-toggle-password]').forEach((button) => {
			const inputId = button.getAttribute('data-toggle-password') || '';
			setPasswordVisible(inputId, false, button);
			button.addEventListener('click', () => {
				const input = document.getElementById(inputId);
				if (!input) {
					return;
				}
				setPasswordVisible(inputId, input.type === 'password', button);
			});
		});

		document.querySelectorAll('.fp-template-manager .fp-template-select').forEach((selectEl) => {
			selectEl.addEventListener('change', async () => {
				syncTemplateSelection(selectEl);
				try {
					await saveConfigPayload({ successMessage: 'Template image selection saved.' });
				} catch (error) {
					showToast('danger', error.message, 5000);
				}
			});
		});

		document.querySelectorAll('.fp-template-manager .fp-template-upload-btn').forEach((button) => {
			button.addEventListener('click', async () => {
				const manager = button.closest('.fp-template-manager');
				try {
					const payload = await uploadTemplateImageFromManager(manager);
					await saveConfigPayload({ quiet: true });
					showToast('success', payload.message || 'Template image uploaded and saved.', 3500);
				} catch (error) {
					showToast('danger', error.message, 5000);
				}
			});
		});

		document.querySelectorAll('[data-config-editor]').forEach((button) => {
			button.addEventListener('click', async () => {
				const scope = button.getAttribute('data-config-editor') || '';
				if (!scope) {
					return;
				}
				try {
					await openConfigEditor(scope);
				} catch (error) {
					showToast('danger', error.message, 5000);
				}
			});
		});

		document.getElementById('config-editor-reload').addEventListener('click', async () => {
			if (!state.configEditorScope) {
				return;
			}
			try {
				await openConfigEditor(state.configEditorScope);
			} catch (error) {
				showToast('danger', error.message, 5000);
			}
		});

		document.getElementById('config-editor-save').addEventListener('click', async () => {
			try {
				await saveConfigEditorContent();
			} catch (error) {
				showToast('danger', error.message, 5000);
			}
		});

		document.getElementById('config-editor-mode-structured').addEventListener('click', () => {
			applyConfigEditorMode('structured');
		});

		document.getElementById('config-editor-mode-raw').addEventListener('click', () => {
			if (isStructuredConfigScope(state.configEditorScope)) {
				syncStructuredRawTextFromObject();
			}
			applyConfigEditorMode('raw');
		});

		document.getElementById('config-editor-add-item').addEventListener('click', () => {
			if (!isCollectionListEditorScope(state.configEditorScope)) {
				return;
			}
			const input = document.getElementById('config-editor-item-input');
			addConfigEditorItem(input.value);
			input.value = '';
			input.focus();
		});

		document.getElementById('config-editor-item-input').addEventListener('keydown', (event) => {
			if (event.key !== 'Enter') {
				return;
			}
			event.preventDefault();
			if (!isCollectionListEditorScope(state.configEditorScope)) {
				return;
			}
			const input = document.getElementById('config-editor-item-input');
			addConfigEditorItem(input.value);
			input.value = '';
		});

		document.getElementById('stats-word-add').addEventListener('click', () => {
			const input = document.getElementById('stats-word-input');
			addStatsWord(input.value);
			input.value = '';
			input.focus();
		});

		document.getElementById('stats-word-input').addEventListener('keydown', (event) => {
			if (event.key !== 'Enter') {
				return;
			}
			event.preventDefault();
			const input = document.getElementById('stats-word-input');
			addStatsWord(input.value);
			input.value = '';
		});

		document.getElementById('stats-exclude-words').addEventListener('input', () => {
			state.statsExcludeWordsItems = normalizeStatsWords(document.getElementById('stats-exclude-words').value);
			renderStatsWordsChips();
		});

		document.getElementById('clean-repl-add').addEventListener('click', () => {
			const findInput = document.getElementById('clean-repl-find');
			const replaceInput = document.getElementById('clean-repl-replace');
			addCleanReplacement(findInput.value, replaceInput.value);
			findInput.value = '';
			replaceInput.value = '';
			findInput.focus();
		});

		document.getElementById('clean-replacements').addEventListener('input', () => {
			state.cleanReplacementItems = parseCleanReplacementsRaw(document.getElementById('clean-replacements').value);
			renderCleanReplacementsList();
		});

		document.getElementById('copy-log').addEventListener('click', async () => {
			const text = document.getElementById('action-log').textContent || '';
			try {
				if (navigator.clipboard && navigator.clipboard.writeText) {
					await navigator.clipboard.writeText(text);
				} else {
					const temp = document.createElement('textarea');
					temp.value = text;
					temp.style.position = 'fixed';
					temp.style.opacity = '0';
					document.body.appendChild(temp);
					temp.focus();
					temp.select();
					document.execCommand('copy');
					document.body.removeChild(temp);
				}
				showToast('success', 'Operation log copied to clipboard.', 2500);
			} catch (error) {
				showToast('danger', 'Unable to copy operation log.', 4500);
			}
		});

		document.getElementById('generate-template-preview').addEventListener('click', async () => {
			await requestTemplatePreview();
		});

		document.getElementById('stop-action').addEventListener('click', async () => {
			try {
				const result = await postJSON('/api/action/stop', {});
				showBanner('warn', result.message || 'Stop requested. Waiting for action to halt.');
				showToast('warning', result.message || 'Stop requested.', 3000);
				startActionPolling();
			} catch (error) {
				showBanner('error', error.message);
				showToast('danger', error.message, 5000);
			}
		});

		document.getElementById('download-output').addEventListener('click', () => {
			const button = document.getElementById('download-output');
			const url = button ? button.dataset.url : '';
			if (!url) {
				showToast('warning', 'No output file is available yet.', 3000);
				return;
			}
			triggerOutputDownload(url);
		});

		document.getElementById('refresh-import-files').addEventListener('click', async () => {
			try {
				await refreshImportFiles(document.getElementById('import-file').value || '');
				showToast('success', 'Import file list refreshed.', 2500);
			} catch (error) {
				showToast('danger', error.message, 5000);
			}
		});

		document.getElementById('upload-import-file').addEventListener('click', async () => {
			try {
				const payload = await uploadImportFile();
				showToast('success', payload.message || 'Import file uploaded.', 3500);
			} catch (error) {
				showToast('danger', error.message, 5000);
			}
		});

    document.getElementById('config-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      try {
		await saveConfigPayload({ successMessage: 'Configuration saved.' });
      } catch (error) {
        setLog(error.message);
        showBanner('error', error.message);
				showToast('danger', error.message, 5000);
      }
    });

		document.getElementById('test-plex-connection').addEventListener('click', async () => {
			try {
				const payload = {
					base_url: document.getElementById('base-url').value,
					token: document.getElementById('token').value,
					retries: Number(document.getElementById('retries').value),
					workers: Number(document.getElementById('workers').value),
					retry_base_ms: Number(document.getElementById('retry-base-ms').value),
					retry_max_ms: Number(document.getElementById('retry-max-ms').value)
				};
				setPlexTestButtonState('loading');
				setLog('Testing Plex connection...');
				const result = await postJSON('/api/plex/test', payload);
				setLog(result.logs || result.message || 'Plex connection succeeded.');
				hideBanner();
				setPlexTestButtonState('success');
				showToast('success', result.message || 'Plex connection succeeded.', 4500);
			} catch (error) {
				setLog(error.message);
				showBanner('error', error.message);
				setPlexTestButtonState('error');
				showToast('danger', error.message, 5000);
			}
		});

    document.getElementById('posters-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('gen-posters', {
        section_keys: Array.from(document.getElementById('poster-sections').selectedOptions).map((option) => option.value),
        upload_posters: document.getElementById('poster-upload').checked,
				missing_posters_only: document.getElementById('poster-missing-only').checked,
        label_types: document.getElementById('poster-label-types').checked,
        trail: document.getElementById('poster-trail').checked
      });
    });

    document.getElementById('clean-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('clean', {
        section_key: document.getElementById('clean-section').value,
        translate: document.getElementById('clean-translate').checked,
        trail: document.getElementById('clean-trail').checked
      });
    });

    document.getElementById('translate-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('translate', {
        section_key: document.getElementById('translate-section').value,
        trail: document.getElementById('translate-trail').checked
      });
    });

    document.getElementById('stats-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('stats', {
        section_key: document.getElementById('stats-section').value
      });
    });

    document.getElementById('label-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('label', {
        section_key: document.getElementById('label-section').value,
        find: document.getElementById('label-find').value,
        add: document.getElementById('label-add').value,
        categories: document.getElementById('label-categories').value,
        update_category: document.getElementById('label-update-category').checked,
        only_category: document.getElementById('label-only-category').checked,
        trail: document.getElementById('label-trail').checked
      });
    });

    document.getElementById('clone-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('clone', {
        section_key: document.getElementById('clone-section').value,
        clone_name: document.getElementById('clone-name').value
      });
    });

    document.getElementById('export-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-export', {
        section_key: document.getElementById('export-section').value,
        coll_file: document.getElementById('export-file').value
      });
    });

    document.getElementById('import-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-import', {
        section_key: document.getElementById('import-section').value,
        coll_file: document.getElementById('import-file').value,
        trail: document.getElementById('import-trail').checked
      });
    });

    document.getElementById('inject-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-inject', {
        section_key: document.getElementById('inject-section').value,
        trail: document.getElementById('inject-trail').checked
      });
    });

    document.getElementById('dupes-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-dupes', {
        section_key: document.getElementById('dupes-section').value
      });
    });

    document.getElementById('delete-non-smart-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-delete-non-smart', {
        section_key: document.getElementById('delete-non-smart-section').value,
        trail: document.getElementById('delete-non-smart-trail').checked
      });
    });

    document.getElementById('path-clean-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('coll-path-clean', {
        section_key: document.getElementById('path-clean-section').value,
        collection_key: document.getElementById('path-clean-collection').value,
        trail: document.getElementById('path-clean-trail').checked
      });
    });

    document.getElementById('backup-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('backup', {});
    });

    document.getElementById('restore-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('restore', {
        restore_file: document.getElementById('restore-file').value,
        trail: document.getElementById('restore-trail').checked
      });
    });

    document.getElementById('rollback-form').addEventListener('submit', (event) => {
      event.preventDefault();
      runAction('rollback', {});
    });

		const mainSwitcher = document.getElementById('main-switcher');
		if (mainSwitcher) {
			mainSwitcher.addEventListener('show', () => {
				updateGlobalSaveButtonVisibility();
			});
			mainSwitcher.addEventListener('shown', () => {
				updateGlobalSaveButtonVisibility();
			});
		}

		const mainTabList = document.querySelector('ul[uk-tab]');
		if (mainTabList) {
			mainTabList.addEventListener('click', () => {
				requestAnimationFrame(updateGlobalSaveButtonVisibility);
			});
		}

		applyActionRowLayout();
		reorderTabsAndActivatePosters();
		updateGlobalSaveButtonVisibility();
		pollActionStatus();
		refreshState();
  </script>
</body>
</html>`))

var webHelpTemplate = template.Must(template.New("help").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.AppName}} Help</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/css/uikit.min.css">
</head>
<body class="fp-help-page">
  <main class="fp-help">
    <div class="fp-help-card">
      <div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-gap-small">
        <div>
		  <p class="uk-text-meta fp-help-kicker">Useful tips</p>
          <h1 class="uk-heading-small uk-margin-small-top">{{.AppName}} web help</h1>
        </div>
        <a class="uk-button uk-button-default" href="/">Back to dashboard</a>
      </div>
      <p class="uk-text-large">The web UI is local-only, uses the same Go workflows as the CLI, and is best treated as an operations console rather than a separate product tier.</p>
    </div>

    <div class="fp-help-card">
      <h2>Recommended flow</h2>
      <ul class="uk-list uk-list-bullet">
        <li>Confirm the Plex URL and token first. A valid config unlocks library discovery and all Plex-backed actions.</li>
        <li>Use dry-run checkboxes before destructive actions such as cleaning titles, deleting non-smart collections, imports, and restores.</li>
        <li>Run stats or duplicate-collection audits before cleanup so you have fresh reports in the output folders.</li>
        <li>Use backup before large operations that touch multiple files or collections.</li>
      </ul>
    </div>

    <div class="fp-help-card">
      <h2>Action notes</h2>
      <ul class="uk-list uk-list-bullet">
        <li><code>Generate posters</code> can target multiple libraries at once and optionally upload finished posters back to Plex.</li>
        <li><code>Clean titles</code> can optionally translate titles first, then apply the configured cleanup replacements.</li>
        <li><code>Path clean</code> rewrites titles from collection item file paths and writes a CSV audit into <code>output/path-clean/</code>.</li>
        <li><code>Export collections</code> writes JSON under <code>output/collections-export/</code> when you provide only a filename.</li>
        <li><code>Restore</code> accepts a partial backup filename; leaving it blank restores the newest archive in non-interactive mode.</li>
      </ul>
    </div>

    <div class="fp-help-card">
      <h2>Troubleshooting</h2>
      <ul class="uk-list uk-list-bullet">
        <li>If the dashboard loads but libraries do not, the config is readable but the Plex connection is failing. Check URL, token, and network reachability.</li>
        <li>If an operation says config validation failed, fix missing template, font, or output paths before retrying that workflow.</li>
        <li>If a restore or rollback fails, review the operation log first, then inspect the latest files under <code>output/restore/</code>.</li>
      </ul>
    </div>
  </main>
</body>
</html>`))
