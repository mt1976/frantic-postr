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
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit.min.js"></script>
  <script defer src="https://cdn.jsdelivr.net/npm/uikit@3.23.8/dist/js/uikit-icons.min.js"></script>
  <style>
    :root {
      --fp-ink: #182126;
      --fp-deep: #23353c;
      --fp-sand: #f5efe2;
      --fp-paper: rgba(255,255,255,0.78);
      --fp-accent: #d36b2d;
      --fp-accent-soft: #f0bc6c;
      --fp-line: rgba(24,33,38,0.12);
      --fp-ok: #2d8b57;
      --fp-warn: #a65b21;
      --fp-error: #9d2f2f;
    }
    html, body {
      min-height: 100%;
      background:
        radial-gradient(circle at top left, rgba(240,188,108,0.4), transparent 30%),
        linear-gradient(160deg, #f8f1e4 0%, #f2e9d8 42%, #ebe5da 100%);
      color: var(--fp-ink);
      font-family: "Space Grotesk", sans-serif;
    }
    .fp-shell {
      max-width: 1380px;
      margin: 0 auto;
      padding: 24px;
    }
    .fp-hero {
      background: linear-gradient(135deg, rgba(35,53,60,0.96), rgba(54,78,85,0.92));
      color: #f7f3ea;
      border-radius: 28px;
      padding: 32px;
      box-shadow: 0 24px 70px rgba(31, 36, 41, 0.16);
      overflow: hidden;
      position: relative;
    }
    .fp-hero::after {
      content: "";
      position: absolute;
      right: -80px;
      top: -40px;
      width: 260px;
      height: 260px;
      background: radial-gradient(circle, rgba(240,188,108,0.46), transparent 70%);
    }
    .fp-kicker, .fp-meta, .fp-log, .uk-input, .uk-select, .uk-textarea, .uk-button {
      font-family: "IBM Plex Mono", monospace;
    }
		.uk-button {
			border-radius: 999px;
			font-weight: 600;
			letter-spacing: 0.02em;
			border: 1px solid rgba(24,33,38,0.16);
			transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease, background 180ms ease, color 180ms ease;
		}
		.uk-button:hover,
		.uk-button:focus {
			transform: translateY(-1px);
			box-shadow: 0 8px 20px rgba(24,33,38,0.14);
			border-color: rgba(24,33,38,0.32);
		}
		.fp-shell .uk-flex.uk-gap-small {
			gap: 10px !important;
		}
		.fp-shell .uk-flex.uk-gap-small.uk-flex-wrap {
			row-gap: 10px !important;
		}
		.uk-button-primary {
			background: linear-gradient(135deg, #2f78d0, #2a62b5);
			color: #f8fbff;
			border-color: rgba(24,54,97,0.72);
		}
		.uk-button-secondary {
			background: rgba(255,255,255,0.72);
			color: var(--fp-deep);
			border-color: rgba(24,33,38,0.24);
		}
		.uk-button-default {
			background: rgba(255,255,255,0.55);
			color: var(--fp-deep);
			border-color: rgba(24,33,38,0.2);
		}
		.uk-button-danger {
			background: linear-gradient(135deg, #bb3f3f, #972f2f);
			color: #fff6f6;
			border-color: rgba(109,30,30,0.8);
		}
		.fp-hero .uk-button-secondary,
		.fp-hero .uk-button-default {
			background: rgba(247,243,234,0.16);
			color: #fff7ea;
			border-color: rgba(247,243,234,0.38);
			box-shadow: 0 10px 22px rgba(10, 14, 17, 0.22);
		}
		.fp-hero .uk-button-secondary:hover,
		.fp-hero .uk-button-default:hover,
		.fp-hero .uk-button-secondary:focus,
		.fp-hero .uk-button-default:focus {
			background: rgba(247,243,234,0.28);
			color: #fffdf7;
			border-color: rgba(247,243,234,0.58);
			box-shadow: 0 14px 26px rgba(10, 14, 17, 0.28);
		}
    .fp-card, .uk-card {
      border-radius: 22px;
      background: var(--fp-paper);
      border: 1px solid var(--fp-line);
      backdrop-filter: blur(10px);
      box-shadow: 0 18px 50px rgba(24,33,38,0.08);
    }
    .fp-title {
      letter-spacing: -0.04em;
      margin: 0;
			color: #ffffff;
    }
    .fp-kicker {
      display: inline-flex;
      gap: 10px;
      align-items: center;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 12px;
      color: rgba(247,243,234,0.82);
    }
    .fp-summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
      gap: 14px;
    }
    .fp-stat {
      padding: 16px 18px;
      border-radius: 18px;
      background: rgba(255,255,255,0.08);
      border: 1px solid rgba(255,255,255,0.08);
    }
    .fp-stat strong {
      display: block;
      font-size: 24px;
      line-height: 1.1;
      margin-top: 10px;
    }
    .fp-section-title {
	  margin: 0 0 10px;
	  font-size: 1.28rem;
	  font-weight: 700;
	  letter-spacing: -0.01em;
    }
		.uk-tab {
			margin-bottom: 8px;
			gap: 8px;
			margin-left: 0;
			padding-left: 0;
		}
		.uk-tab > * {
			padding-left: 0;
		}
		.uk-switcher {
			margin-left: 0;
			padding-left: 0;
		}
		.uk-tab > * > a {
			border: 1px solid rgba(24,33,38,0.14);
			border-radius: 12px;
			background: rgba(255,255,255,0.62);
			color: rgba(24,33,38,0.8);
			text-transform: none;
			letter-spacing: 0.02em;
			font-size: 13px;
			font-weight: 600;
			padding: 10px 14px;
			transition: border-color 180ms ease, background 180ms ease, color 180ms ease, transform 180ms ease;
		}
		.uk-tab > * > a:hover {
			border-color: rgba(24,33,38,0.34);
			background: rgba(255,255,255,0.85);
			color: var(--fp-ink);
			transform: translateY(-1px);
		}
		.uk-tab > .uk-active > a {
			border-color: rgba(211,107,45,0.48);
			background: linear-gradient(135deg, rgba(211,107,45,0.2), rgba(240,188,108,0.25));
			color: var(--fp-deep);
			box-shadow: 0 8px 22px rgba(35,53,60,0.1);
		}
    .fp-banner {
      border-radius: 18px;
      padding: 14px 16px;
      display: none;
    }
    .fp-banner.active {
      display: block;
    }
    .fp-banner.ok { background: rgba(45,139,87,0.12); color: var(--fp-ok); }
    .fp-banner.warn { background: rgba(166,91,33,0.14); color: var(--fp-warn); }
    .fp-banner.error { background: rgba(157,47,47,0.12); color: var(--fp-error); }
    .fp-log {
      min-height: 280px;
      max-height: 480px;
      overflow: auto;
      padding: 18px;
      border-radius: 18px;
      background: #151d21;
      color: #e9e6dd;
      white-space: pre-wrap;
      border: 1px solid rgba(255,255,255,0.07);
    }
		.fp-progress-wrap {
			border-radius: 14px;
			border: 1px solid var(--fp-line);
			background: rgba(255,255,255,0.6);
			padding: 12px;
		}
		.fp-progress-head {
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 10px;
			font-family: "IBM Plex Mono", monospace;
			font-size: 12px;
			color: rgba(24,33,38,0.78);
			margin-bottom: 8px;
		}
		.fp-progress-title {
			margin: 0;
			font-size: 13px;
			letter-spacing: 0.04em;
			text-transform: uppercase;
		}
		.fp-progress-bar {
			width: 100%;
			height: 14px;
			appearance: none;
			border: 0;
			border-radius: 999px;
			overflow: hidden;
			background: rgba(24,33,38,0.1);
		}
		.fp-progress-bar::-webkit-progress-bar {
			background: rgba(24,33,38,0.1);
			border-radius: 999px;
		}
		.fp-progress-bar::-webkit-progress-value {
			background: linear-gradient(90deg, var(--fp-accent), var(--fp-accent-soft));
			border-radius: 999px;
			transition: width 120ms linear;
		}
		.fp-progress-bar::-moz-progress-bar {
			background: linear-gradient(90deg, var(--fp-accent), var(--fp-accent-soft));
			border-radius: 999px;
		}
		.fp-progress-meta {
			margin-top: 8px;
			font-family: "IBM Plex Mono", monospace;
			font-size: 12px;
			color: rgba(24,33,38,0.66);
		}
    .fp-chip {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      padding: 5px 10px;
      background: rgba(211,107,45,0.12);
      color: var(--fp-accent);
      font-size: 12px;
      margin: 4px 6px 0 0;
    }
    .fp-actions-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
	  gap: 22px;
			align-items: start;
    }
		.uk-switcher > li {
			margin-top: 4px;
			padding-top: 4px;
		}
		.uk-switcher > li > .fp-actions-grid,
		.uk-switcher > li > .uk-card {
			margin-bottom: 10px;
		}
    .fp-muted {
      color: rgba(24,33,38,0.68);
    }
    .fp-path {
      word-break: break-all;
    }
    .fp-footer-note {
      font-size: 12px;
      color: rgba(24,33,38,0.62);
    }
		.fp-test-button {
			position: relative;
			border: 1px solid rgba(24,33,38,0.16);
			transition: border-color 180ms ease, box-shadow 180ms ease, color 180ms ease, background 180ms ease;
		}
		.fp-test-button:hover,
		.fp-test-button:focus {
			border-color: rgba(24,33,38,0.32);
		}
		.fp-test-button.is-loading {
			border-color: rgba(211,107,45,0.6);
			box-shadow: 0 0 0 3px rgba(211,107,45,0.12);
		}
		.fp-test-button.is-success {
			border-color: rgba(45,139,87,0.9);
			box-shadow: 0 0 0 3px rgba(45,139,87,0.14);
			color: var(--fp-ok);
			background: rgba(45,139,87,0.08);
		}
		.fp-test-button.is-error {
			border-color: rgba(157,47,47,0.9);
			box-shadow: 0 0 0 3px rgba(157,47,47,0.14);
			color: var(--fp-error);
			background: rgba(157,47,47,0.08);
		}
		.fp-test-button-icon {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			width: 1.1rem;
			margin-left: 8px;
			opacity: 0;
			transform: scale(0.8);
			transition: opacity 180ms ease, transform 180ms ease;
		}
		.fp-test-button.is-success .fp-test-button-icon,
		.fp-test-button.is-error .fp-test-button-icon,
		.fp-test-button.is-loading .fp-test-button-icon {
			opacity: 1;
			transform: scale(1);
		}
		.fp-test-button.is-success .fp-test-button-icon {
			color: var(--fp-ok);
		}
		.fp-test-button.is-error .fp-test-button-icon {
			color: var(--fp-error);
		}
		.fp-test-button.is-loading .fp-test-button-icon {
			color: var(--fp-accent);
			animation: fp-pulse 1s ease-in-out infinite;
		}
		@keyframes fp-pulse {
			0%, 100% { opacity: 0.45; }
			50% { opacity: 1; }
		}
    @media (max-width: 640px) {
      .fp-shell { padding: 14px; }
      .fp-hero { padding: 22px; border-radius: 22px; }
      .fp-log { min-height: 220px; }
    }
  </style>
</head>
<body>
  <div class="fp-shell">
    <section class="fp-hero uk-margin-large-bottom uk-animation-slide-top-small">
      <div class="uk-flex uk-flex-between uk-flex-wrap uk-flex-middle uk-gap-small">
        <div>
          <div class="fp-kicker">{{.Mode}} <span>•</span> Version {{.Version}}</div>
          <h1 class="fp-title uk-heading-small">{{.AppName}} control room</h1>
          <p class="uk-text-large uk-margin-small-top uk-margin-medium-bottom" style="max-width: 760px; color: rgba(247,243,234,0.86);">Run poster generation, library cleanup, collection maintenance, and backup workflows from a local UIKit dashboard without losing the existing Plex-aware Go logic.</p>
          <div class="uk-flex uk-flex-wrap uk-gap-small">
						<button class="uk-button uk-button-secondary" id="refresh-state"><span uk-icon="refresh" class="uk-margin-small-right"></span>Refresh state</button>
						<a class="uk-button uk-button-default" href="{{.HelpPath}}"><span uk-icon="question" class="uk-margin-small-right"></span>Help & tips</a>
						<button class="uk-button uk-button-default" uk-toggle="target: #about-modal"><span uk-icon="info" class="uk-margin-small-right"></span>About</button>
          </div>
        </div>
        <div class="fp-summary-grid" style="min-width: min(100%, 380px); max-width: 420px;">
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
		<ul uk-tab>
			<li><a href="#"><span uk-icon="settings" class="uk-margin-small-right"></span>Config</a></li>
			<li><a href="#"><span uk-icon="image" class="uk-margin-small-right"></span>Posters</a></li>
			<li><a href="#"><span uk-icon="folder" class="uk-margin-small-right"></span>Library</a></li>
			<li><a href="#"><span uk-icon="thumbnails" class="uk-margin-small-right"></span>Collections</a></li>
			<li><a href="#"><span uk-icon="lock" class="uk-margin-small-right"></span>Safety</a></li>
		</ul>
		<ul class="uk-switcher uk-margin">
			<li>
				<div class="fp-actions-grid">
					<div class="uk-card uk-card-body fp-card uk-animation-slide-left-small">
						<h2 class="fp-section-title">Configuration</h2>
						<p class="fp-muted uk-margin-small-top">If you do not pass <code>--config</code>, the web UI uses <code>config/config.toml</code>. Saving here writes the exposed settings back to the active config files, and keeps Plex secrets in <code>plex_config</code> when you use a split file.</p>
						<form id="config-form" class="uk-grid-small" uk-grid>
            <div class="uk-width-1-1">
              <label class="uk-form-label" for="base-url">Plex URL</label>
              <input class="uk-input" id="base-url" name="base_url" type="url" required>
            </div>
            <div class="uk-width-1-1">
              <label class="uk-form-label" for="token">API key / token</label>
			  <input class="uk-input" id="token" name="token" type="text" autocomplete="off">
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
							<label class="uk-form-label" for="output-dir-input">Output directory</label>
							<input class="uk-input" id="output-dir-input" name="output_dir" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="log-file-input">Log file</label>
							<input class="uk-input" id="log-file-input" name="log_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-endpoint">Translate endpoint</label>
							<input class="uk-input" id="translate-endpoint" name="translate_endpoint" type="url">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-api-key">Translate API key</label>
							<input class="uk-input" id="translate-api-key" name="translate_api_key" type="password" autocomplete="off">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="translate-rate-limit">Translate rate limit per minute</label>
							<input class="uk-input" id="translate-rate-limit" name="translate_rate_limit_per_minute" type="number" min="1">
						</div>
						<div class="uk-width-1-1">
							<h3 class="fp-section-title uk-margin-small-top">Config files and templates</h3>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="template-image">Template image</label>
							<input class="uk-input" id="template-image" name="template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="type-template-image">Type template image</label>
							<input class="uk-input" id="type-template-image" name="type_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="studio-template-image">Studio template image</label>
							<input class="uk-input" id="studio-template-image" name="studio_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="admin-template-image">Admin template image</label>
							<input class="uk-input" id="admin-template-image" name="admin_template_image" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="type-collections-file">Type collections file</label>
							<input class="uk-input" id="type-collections-file" name="type_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="studio-collections-file">Studio collections file</label>
							<input class="uk-input" id="studio-collections-file" name="studio_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="admin-collections-file">Admin collections file</label>
							<input class="uk-input" id="admin-collections-file" name="admin_collections_file" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="plex-config-file">Plex config file</label>
							<input class="uk-input" id="plex-config-file" name="plex_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="label-config-file">Label config file</label>
							<input class="uk-input" id="label-config-file" name="label_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="collection-config-file">Collection config file</label>
							<input class="uk-input" id="collection-config-file" name="collection_config" type="text">
						</div>
						<div class="uk-width-1-1">
							<h3 class="fp-section-title uk-margin-small-top">Cleaning, stats, backup, and font</h3>
						</div>
						<div class="uk-width-1-1">
							<label><input class="uk-checkbox" id="translate-to-english" type="checkbox"> Translate to English in clean config</label>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="clean-replacements">Clean replacements</label>
							<textarea class="uk-textarea" id="clean-replacements" rows="6" placeholder="& =  and 
		FULL MOVIE =  "></textarea>
						</div>
						<div class="uk-width-1-1">
							<label class="uk-form-label" for="stats-exclude-words">Stats exclude words</label>
							<textarea class="uk-textarea" id="stats-exclude-words" rows="4" placeholder="comma,separated,words"></textarea>
						</div>
						<div class="uk-width-1-2">
							<label class="uk-form-label" for="backup-retention-days">Backup retention days</label>
							<input class="uk-input" id="backup-retention-days" type="number" min="0">
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
		            <div class="uk-width-1-1 uk-flex uk-flex-between uk-flex-middle uk-margin-small-top uk-flex-wrap uk-gap-small">
		              <span class="fp-footer-note">The server stays local to 127.0.0.1.</span>
		              <div class="uk-flex uk-gap-small uk-flex-wrap">
								<button class="uk-button uk-button-default fp-test-button" id="test-plex-connection" type="button">Test Plex connection <span class="fp-test-button-icon" id="test-plex-connection-icon" aria-hidden="true"></span></button>
		                <button class="uk-button uk-button-primary" type="submit"><span uk-icon="check" class="uk-margin-small-right"></span>Save config</button>
		              </div>
		            </div>
						</form>
					</div>

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
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run translation</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Stats and labels</h3>
                <form id="stats-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="stats-section">Stats library</label>
                  <select class="uk-select" id="stats-section"></select>
									<button class="uk-button uk-button-secondary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Run stats</button>
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
									<button class="uk-button uk-button-secondary uk-margin-top" type="submit"><span uk-icon="download" class="uk-margin-small-right"></span>Export collections</button>
                </form>
                <hr>
                <form id="import-form">
                  <label class="uk-form-label" for="import-section">Import target library</label>
                  <select class="uk-select" id="import-section"></select>
									<label class="uk-form-label uk-margin-small-top" for="import-file">Import file</label>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle">
										<select class="uk-select" id="import-file" style="min-width: 320px;"></select>
										<button class="uk-button uk-button-default" id="refresh-import-files" type="button"><span uk-icon="refresh" class="uk-margin-small-right"></span>Refresh files</button>
									</div>
									<div class="uk-flex uk-gap-small uk-flex-wrap uk-flex-middle uk-margin-small-top">
										<input class="uk-input" id="import-file-upload" type="file" accept=".json,application/json" style="min-width: 320px;">
										<button class="uk-button uk-button-secondary" id="upload-import-file" type="button"><span uk-icon="upload" class="uk-margin-small-right"></span>Upload import file</button>
									</div>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="import-trail" type="checkbox"> Dry run only</label>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="upload" class="uk-margin-small-right"></span>Import collections</button>
                </form>
              </div>
              <div class="uk-card uk-card-body fp-card">
                <h3 class="fp-section-title">Collection maintenance</h3>
                <form id="inject-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="inject-section">Inject target library</label>
                  <select class="uk-select" id="inject-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="inject-trail" type="checkbox"> Dry run only</label>
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="plus" class="uk-margin-small-right"></span>Inject configured collections</button>
                </form>
                <form id="dupes-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="dupes-section">Duplicates library</label>
                  <select class="uk-select" id="dupes-section"></select>
									<button class="uk-button uk-button-secondary uk-margin-top" type="submit"><span uk-icon="search" class="uk-margin-small-right"></span>Audit duplicates</button>
                </form>
                <form id="delete-non-smart-form" class="uk-margin-small-bottom">
                  <label class="uk-form-label" for="delete-non-smart-section">Delete non-smart library</label>
                  <select class="uk-select" id="delete-non-smart-section"></select>
                  <label class="uk-margin-small-top"><input class="uk-checkbox" id="delete-non-smart-trail" type="checkbox"> Dry run only</label>
									<button class="uk-button uk-button-danger uk-margin-top" type="submit"><span uk-icon="trash" class="uk-margin-small-right"></span>Delete non-smart collections</button>
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
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="play" class="uk-margin-small-right"></span>Path clean collection</button>
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
									<button class="uk-button uk-button-primary uk-margin-top" type="submit"><span uk-icon="history" class="uk-margin-small-right"></span>Restore backup</button>
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
				<div class="uk-flex uk-flex-wrap uk-gap-small uk-flex-middle">
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
					<pre id="action-log" class="fp-log uk-margin-top">No action has been run yet.</pre>
				</div>
			</li>
		</ul>
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
		plexTestResetTimer: null,
		actionPollTimer: null,
		actionPollBusy: false,
		actionPollFast: 300,
		actionPollSlow: 1200,
		lastActionStatusHash: '',
		lastDownloadToken: '',
		runtimeStatusToastShown: false,
		lastParsedLogText: '',
		seenToastLogLines: {}
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

    function setLog(text) {
      document.getElementById('action-log').textContent = text || 'No output returned.';
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

		function moveConfigTabLast() {
			const tabList = document.querySelector('ul[uk-tab]');
			if (!tabList || !tabList.children || tabList.children.length < 2) {
				return;
			}
			const switcher = tabList.nextElementSibling;
			if (!switcher || !switcher.classList || !switcher.classList.contains('uk-switcher') || switcher.children.length < 2) {
				return;
			}
			const firstTab = tabList.children[0];
			const firstPanel = switcher.children[0];
			if (!firstTab || !firstPanel) {
				return;
			}
			tabList.appendChild(firstTab);
			switcher.appendChild(firstPanel);
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
        const payload = {
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
        const result = await postJSON('/api/config', payload);
        setLog(result.logs || result.message || 'Configuration updated.');
        showBanner('ok', result.message || 'Configuration saved.');
				showToast('success', result.message || 'Configuration saved.', 3500);
        await refreshState();
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
				showBanner('ok', result.message || 'Plex connection succeeded.');
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

		moveConfigTabLast();
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
  <style>
    body {
      background: linear-gradient(180deg, #f6eddc 0%, #f1e7d7 100%);
      color: #182126;
      font-family: "Space Grotesk", sans-serif;
      margin: 0;
    }
    .fp-help {
      max-width: 980px;
      margin: 0 auto;
      padding: 28px 18px 40px;
    }
    .fp-help-card {
      border-radius: 24px;
      background: rgba(255,255,255,0.8);
      padding: 28px;
      box-shadow: 0 18px 50px rgba(24,33,38,0.08);
      border: 1px solid rgba(24,33,38,0.1);
      margin-bottom: 18px;
    }
    code { font-family: "IBM Plex Mono", monospace; }
  </style>
</head>
<body>
  <main class="fp-help">
    <div class="fp-help-card">
      <div class="uk-flex uk-flex-between uk-flex-middle uk-flex-wrap uk-gap-small">
        <div>
          <p class="uk-text-meta" style="letter-spacing: 0.16em; text-transform: uppercase;">Useful tips</p>
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
