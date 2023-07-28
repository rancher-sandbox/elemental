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

import '~/support/commands';
import 'cypress-file-upload';
import filterTests from '~/support/filterTests.js';
import * as utils from "~/support/utils";
import * as cypressLib from '@rancher-ecp-qa/cypress-library';
import { qase } from 'cypress-qase-reporter/dist/mocha';


Cypress.config();
describe('Upgrade tests', () => {
  const channelName   = "mychannel"
  const clusterName   = "mycluster"
  const elementalUser = "elemental-user"
  const uiAccount     = Cypress.env('ui_account');
  const uiPassword    = "rancherpassword"
  const upgradeImage  = Cypress.env('upgrade_image')

  beforeEach(() => {
    (uiAccount == "user") ? cy.login(elementalUser, uiPassword) : cy.login();
    cy.visit('/');

    // Open the navigation menu
    cypressLib.burgerMenuOpenIfClosed();    

    // Click on the Elemental's icon
    cypressLib.accesMenu('OS Management');
  });

  filterTests(['upgrade'], () => {
    qase(33,
      it('Check OS Versions', () => {
        cy.clickNavMenu(["Advanced", "OS Versions"]);
        utils.isOperatorVersion('dev') ? cy.contains('Active latest-dev', {timeout: 120000}): null;
        utils.isOperatorVersion('staging') ? cy.contains('Active latest-staging', {timeout: 120000}): null;
      })
    );

    qase(34,
      it('Upgrade one node (different methods if rke2 or k3s)', () => {
        cypressLib.checkClusterStatus(clusterName, 'Active', 600000);
        cypressLib.burgerMenuOpenIfClosed();
        cypressLib.accesMenu('OS Management');
        /////////////////////////////////////////
        // K3s cluster upgraded with OS Image
        // RKE2 cluster upgraded with OS version channel
        cy.clickNavMenu(["Advanced", "Update Groups"]);
        cy.getBySel('masthead-create')
        .contains('Create')
          .click();
        cy.get('.primaryheader')
          .contains('Update Group: Create');
        cy.getBySel('name-ns-description-name')
          .type(channelName);
        cy.contains('Target Cluster')
        cy.getBySel('cluster-target')
          .click();
        cy.contains(clusterName)
          .click();
        if (utils.isK8sVersion("k3s")) {
          cy.getBySel('upgrade-choice-selector')
            .parent()
            .contains('Use image from registry')
            .click();
          cy.getBySel('os-image-box')
            .type(upgradeImage)
        } else {
          cy.getBySel('upgrade-choice-selector')
            .parent()
            .contains('Use Managed OS Version')
            .click();
          cy.getBySel('os-version-box')
            .click()
          cy.getBySel('os-version-box')
            .parents()
            .contains('dev')
            .click();
        }

        cy.getBySel('form-save')
        .contains('Create')
          .click();
        // Status changes a lot right after the creation so let's wait 10 secondes
        // before checking
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy.wait(10000);
        cy.getBySel('sortable-cell-0-0')
          .contains('Active');

        // Workaround to avoid sporadic issue with Upgrade
        // https://github.com/rancher/elemental/issues/410
        // Restart fleet agent inside downstream cluster
        cypressLib.burgerMenuOpenIfClosed();
        cy.getBySel('side-menu')
          .contains(clusterName)
          .click();
        cy.contains('Workload')
          .click();
        cy.contains('Pods')
          .click();
        cy.get('.header-buttons > :nth-child(2)')
          .click();
        // eslint-disable-next-line cypress/no-unnecessary-waiting
        cy.wait(20000);
        cy.get('.shell-body')
          .type('kubectl scale deployment/fleet-agent -n cattle-fleet-system --replicas=0{enter}');
        cy.get('.shell-body')
          .type('kubectl scale deployment/fleet-agent -n cattle-fleet-system --replicas=1{enter}');

        // Check if the node reboots to apply the upgrade
        cypressLib.burgerMenuOpenIfClosed();    
        cypressLib.accesMenu('OS Management');
        cy.clickNavMenu(["Dashboard"]);
        cy.getBySel('card-clusters')
          .contains('Manage Elemental Clusters')
          .click()
        cy.get('.title')
          .contains('Clusters');
        cy.contains(clusterName)
          .click();
        cy.get('.primaryheader')
          .contains('Active');
        cy.get('.primaryheader')
          .contains('Active', {timeout: 420000}).should('not.exist');
        cy.get('.primaryheader')
          .contains('Active', {timeout: 420000});
      })
    );

    qase(35,
      it('Cannot create two upgrade groups targeting the same cluster', () => {
        cy.clickNavMenu(["Advanced", "Update Groups"]);
        cy.getBySel('masthead-create')
        .contains('Create')
          .click();
        cy.get('.primaryheader')
          .contains('Update Group: Create');
        cy.getBySel('cluster-target')
          .click();
        cy.contains('Sorry, no matching options');
      })
    );

    qase(37,
      it('Delete OS Versions Channels', () => {
        cy.clickNavMenu(["Advanced", "OS Version Channels"]);
        cy.deleteAllResources();
        cy.clickNavMenu(["Advanced", "OS Versions"]);
        cy.contains('There are no rows to show');
      })
    );
  })
})
