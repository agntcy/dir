export type CatalogHeaderStats = {
  count: number;
  catalogHydrating: boolean;
  hydrationError: string;
};

class HeaderStatsState {
  stats = $state<CatalogHeaderStats | null>(null);

  set(stats: CatalogHeaderStats | null) {
    this.stats = stats;
  }
}

export const headerStatsState = new HeaderStatsState();
