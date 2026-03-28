// HackForger frontend module entry point.
// Lazy-loads Vue components when their mount elements are present.

export function initHackforger() {
  // BountyPanel
  const bountyEl = document.getElementById('hackforger-bounty-panel');
  if (bountyEl) {
    (async () => {
      const {default: BountyPanel} = await import(
        /* webpackChunkName: "hackforger-bounty" */
        '../../components/hackforger/BountyPanel.vue'
      );
      const {createApp} = await import('vue');
      createApp(BountyPanel, {
        bountyId: bountyEl.getAttribute('data-bounty-id'),
      }).mount(bountyEl);
    })();
  }
}
