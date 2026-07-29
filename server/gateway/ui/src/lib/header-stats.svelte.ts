export type CatalogHeaderStats = {
  totalCount: number;
};

class HeaderStatsState {
  stats = $state<CatalogHeaderStats | null>(null);

  set(stats: CatalogHeaderStats | null) {
    this.stats = stats;
  }
}

export const headerStatsState = new HeaderStatsState();
