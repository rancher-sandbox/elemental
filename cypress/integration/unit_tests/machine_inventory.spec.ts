import { TopLevelMenu } from '~/cypress/support/toplevelmenu';
import '~/cypress/support/functions';
import { Elemental } from '../../support/elemental';

Cypress.config();
describe('Machine inventory testing', () => {
  const topLevelMenu = new TopLevelMenu();
  const elemental    = new Elemental();
  const k8s_version  = Cypress.env('k8s_version');

  beforeEach(() => {
    cy.login();
    cy.visit('/');

    // Open the navigation menu
    topLevelMenu.openIfClosed();

    // Click on the Elemental's icon
    elemental.accessElementalMenu(); 
  });

  it('Check that machine inventory has been created', () => {
    cy.contains('Manage Machine Inventories').click();
    cy.contains('.badge-state', 'Active').should('exist');
    cy.contains('Namespace: fleet-default').should('exist');
  });

  it('Create Elemental cluster', () => {
    cy.contains('Create Elemental Cluster').click();
    cy.typeValue({label: 'Cluster Name', value: 'myelementalcluster'});
    cy.typeValue({label: 'Cluster Description', value: 'My Elemental testing cluster'});
    cy.contains('Kubernetes Version').click();
    cy.contains(k8s_version).click();
    cy.clickButton('Create');
    cy.contains('Cluster: myelementalcluster', {timeout: 20000});
    cy.contains('Cluster: myelementalcluster Active', {timeout: 360000});
  });

  it('Check Elemental cluster status', () => {
    topLevelMenu.openIfClosed();
    cy.contains('Home').click();
    // The new cluster must be in active state
    cy.get('[data-node-id="fleet-default/myelementalcluster"]').contains('Active');
    // Go into the dedicated cluster page
    topLevelMenu.openIfClosed();
    cy.contains('myelementalcluster').click();
  })
});
