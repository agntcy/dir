export const DEFAULT_CATALOG_TITLE = 'AI Catalog';

class CatalogTitleState {
	title = $state(DEFAULT_CATALOG_TITLE);
	loaded = $state(false);

	async load() {
		try {
			const response = await fetch('/ui/config.json');
			if (!response.ok) return;

			const data = (await response.json()) as { catalogTitle?: unknown };
			if (typeof data.catalogTitle === 'string') {
				const trimmed = data.catalogTitle.trim();
				if (trimmed) this.title = trimmed;
			}
		} catch {
			// Keep the built-in default when config is unavailable (e.g. Vite dev without proxy).
		} finally {
			this.loaded = true;
		}
	}
}

export const catalogTitleState = new CatalogTitleState();
