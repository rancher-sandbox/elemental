/*
Copyright © 2022 - 2023 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import { TopLevelMenu } from '~/support/toplevelmenu';
import '~/support/commands';
import filterTests from '~/support/filterTests.js';

filterTests(['main'], () => {
  Cypress.config();
  describe('Deploy application in fresh Elemental Cluster', () => {
    const topLevelMenu = new TopLevelMenu();
    const clusterName  = "mycluster"
    beforeEach(() => {
      cy.login();
      cy.visit('/');
    });
    
    it('Deploy Alerting Drivers application', () => {
      // Unfortunately, this wait is needed mainly with RKE2 because the cluster
      // is switching status and it is hard to catch in automated way...
      cy.wait(180000);
      topLevelMenu.openIfClosed();
      cy.contains('Home')
        .click();
      cy.get('[data-node-id="fleet-default/'+clusterName+'"]')
        .contains('Active',  {timeout: 300000});
      cy.contains(clusterName)
        .click();
      cy.contains('Apps')
        .click();
      cy.contains('Charts')
        .click();
      cy.contains('Alerting Drivers', {timeout:30000})
        .click();
      cy.contains('.name-logo-install', 'Alerting Drivers', {timeout:30000});
      cy.clickButton('Install');
      cy.contains('.outer-container > .header', 'Alerting Drivers');
      cy.clickButton('Next');
      cy.clickButton('Install');
      cy.contains('SUCCESS: helm install', {timeout:120000});
      cy.reload;
      cy.contains('Deployed rancher-alerting-drivers');
    });
  
    it('Remove Alerting Drivers application', () => {
      topLevelMenu.openIfClosed();
      cy.contains(clusterName)
        .click();
      cy.contains('Apps')
        .click();
      cy.contains('Installed Apps')
        .click();
      cy.contains('.title', 'Installed Apps', {timeout:20000});
      cy.get('[width="30"] > .checkbox-outer-container')
        .click();
      cy.clickButton('Delete');
      cy.confirmDelete();
      cy.contains('SUCCESS: helm uninstall', {timeout:60000});
      cy.contains('.apps', 'rancher-alerting-drivers')
        .should('not.exist');
    });
  });
});
