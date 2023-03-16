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

import 'cypress-file-upload';

// Global
interface hardwareLabels {
  [key: string]: string;
};

const hwLabels: hardwareLabels = {
  'CPUModel': '${System Data/CPU/Model}',
  'CPUVendor': '${System Data/CPU/Vendor}',
  'NumberBlockDevices': '${System Data/Block Devices/Number Devices}',
  'NumberNetInterface': '${System Data/Network/Number Interfaces}',
  'CPUVendorTotalCPUCores': '${System Data/CPU/Total Cores}',
  'TotalCPUThread': '${System Data/CPU/Total Threads}',
  'TotalMemory': '${System Data/Memory/Total Physical Bytes}'
};

// Generic functions

// Log into Rancher
Cypress.Commands.add('login', (
  username = Cypress.env('username'),
  password = Cypress.env('password'),
  cacheSession = Cypress.env('cache_session')) => {
    const login = () => {
      let loginPath ="/v3-public/localProviders/local*";
      cy.intercept('POST', loginPath).as('loginReq');
      
      cy.visit('/auth/login');

      cy.get('[data-testid="local-login-username"]')
        .type(username, {log: false});

      cy.get('[data-testid="local-login-password"]')
        .type(password, {log: false});

      cy.get('[data-testid="login-submit"]')
        .click();
      cy.wait('@loginReq');
      cy.get('[data-testid="banner-title"]').contains('Welcome to Rancher');
      } 

    if (cacheSession) {
      cy.session([username, password], login);
    } else {
      login();
    }
});

// Search fields by label
Cypress.Commands.add('byLabel', (label) => {
  cy.get('.labeled-input')
    .contains(label)
    .siblings('input');
});

// Search button by label
Cypress.Commands.add('clickButton', (label) => {
  cy.get('.btn')
    .contains(label)
    .click();
});

// Confirm the delete operation
Cypress.Commands.add('confirmDelete', () => {
  cy.get('[data-testid="prompt-remove-confirm-button"]')
    .click()
});

// Make sure we are in the desired menu inside a cluster (local by default)
// You can access submenu by giving submenu name in the array
// ex:  cy.clickClusterMenu(['Menu', 'Submenu'])
Cypress.Commands.add('clickNavMenu', (listLabel: string[]) => {
  listLabel.forEach(label => cy.get('nav').contains(label).click());
});

// Insert a value in a field *BUT* force a clear before!
Cypress.Commands.add('typeValue', ({label, value, noLabel, log=true}) => {
  if (noLabel === true) {
    cy.get(label)
      .focus()
      .clear()
      .type(value, {log: log});
  } else {
    cy.byLabel(label)
      .focus()
      .clear()
      .type(value, {log: log});
  }
});

// Make sure we are in the desired menu inside a cluster (local by default)
// You can access submenu by giving submenu name in the array
// ex:  cy.clickClusterMenu(['Menu', 'Submenu'])
Cypress.Commands.add('clickClusterMenu', (listLabel: string[]) => {
  listLabel.forEach(label => cy.get('nav').contains(label).click());
});

// Insert a key/value pair
Cypress.Commands.add('typeKeyValue', ({key, value}) => {
  cy.get(key)
    .clear()
    .type(value);
});

Cypress.Commands.overwrite('type', (originalFn, subject, text, options = {}) => {
  options.delay = 100;
  return originalFn(subject, text, options);
});

// Add a delay between command without using cy.wait()
// https://github.com/cypress-io/cypress/issues/249#issuecomment-443021084
const COMMAND_DELAY = 1000;

for (const command of ['visit', 'click', 'trigger', 'type', 'clear', 'reload', 'contains']) {
    Cypress.Commands.overwrite(command, (originalFn, ...args) => {
        const origVal = originalFn(...args);

        return new Promise((resolve) => {
            setTimeout(() => {
                resolve(origVal);
            }, COMMAND_DELAY);
        });
    });
}; 

// Add Helm repo
Cypress.Commands.add('addHelmRepo', ({repoName, repoUrl, repoType}) => {
  cy.clickClusterMenu(['Apps', 'Repositories'])

  // Make sure we are in the 'Repositories' screen (test failed here before)
  cy.contains('header', 'Repositories')
    .should('be.visible');
  cy.contains('Create')
    .should('be.visible');

  cy.clickButton('Create');
  cy.contains('Repository: Create')
    .should('be.visible');
  cy.typeValue({label: 'Name', value: repoName});
  if (repoType === 'git') {
    cy.contains('Git repository')
      .click();
    cy.typeValue({label: 'Git Repo URL', value: repoUrl});
    cy.typeValue({label: 'Git Branch', value: 'main'});
  } else {
    cy.typeValue({label: 'Index URL', value: repoUrl});
  }
  cy.clickButton('Create');
});

// Delete all resources from a page
Cypress.Commands.add('deleteAllResources', () => {  
  cy.get('[width="30"] > .checkbox-outer-container')
    .click();
  cy.contains('[data-testid="sortable-table-promptRemove"]', 'Delete')
    .click()
  cy.confirmDelete();
  // Sometimes, UI is crashing when a resource is deleted
  // A reload should workaround the failure
  cy.get('body').then(($body) => {
    if (!$body.text().includes('There are no rows to show.')) {
        cy.reload();
        cy.log('RELOAD TRIGGERED');
        cy.screenshot('reload-triggered');
      };
    });
  cy.contains('There are no rows to show', {timeout: 15000});
});

// Machine registration functions

// Create a machine registration
Cypress.Commands.add('createMachReg', ({
  machRegName,
  namespace='fleet-default',
  checkLabels=false,
  checkAnnotations=false,
  checkInventoryLabels=false,
  checkInventoryAnnotations=false,
  customCloudConfig='',
  checkDefaultCloudConfig=true }) => {
    cy.clickNavMenu(["Dashboard"]);
    cy.get('[data-testid="button-create-registration-endpoint"]')
      .click();
    cy.get('[data-testid="name-ns-description-name"]')
      .type(machRegName);

    if (customCloudConfig != '') {
      cy.get('input[type="file"]')
        .attachFile({filePath: customCloudConfig});
    }

    if (checkLabels) {
      cy.addMachRegLabel({labelName: 'myLabel1', labelValue: 'myLabelValue1'});
      }

    if (checkAnnotations) {
      cy.addMachRegAnnotation({
        annotationName: 'myAnnotation1',
        annotationValue: 'myAnnotationValue1'});
    }

    if (checkInventoryLabels) {
      cy.addMachInvLabel({
        labelName: 'myInvLabel1',
        labelValue: 'myInvLabelValue1'});
    }

    if (checkInventoryAnnotations) {
      cy.addMachInvAnnotation({
        annotationName: 'myInvAnnotation1',
        annotationValue: 'myInvAnnotationValue1'});
    }

    cy.contains('[data-testid="form-save"]', 'Create')
      .click();

    // Make sure the machine registration is created and active
    cy.contains('.masthead', 'Registration Endpoint: '+ machRegName + 'Active')
      .should('exist');

    // Check the namespace
    cy.contains('.masthead', 'Namespace: '+ namespace)
      .should('exist');

    // Make sure there is an URL registration in the Registration URL block
    cy.contains('[data-testid="registration-url"]', /https:\/\/.*elemental\/registration/);

    // Try to download the registration file and check it
    cy.get('[data-testid="download-btn"]')
      .click();
    cy.verifyDownload(machRegName + '_registrationURL.yaml');
    cy.contains('Saving')
      .should('not.exist');
  
      // Check Cloud configuration
    // TODO: Maybe the check may be improved in one line
    if (checkDefaultCloudConfig) {
      cy.get('[data-testid="yaml-editor-code-mirror"]')
        .should('include.text','config:')
        .should('include.text','cloud-config:')
        .should('include.text','users:')
        .should('include.text','- name: root')
        .should('include.text','passwd: root')
        .should('include.text','elemental:')
        .should('include.text','install:')
        .should('include.text','device: /dev/nvme0n1')
        .should('include.text','poweroff: true');
    }

    // Check label and annotation in YAML
    // For now, we can only check in YAML because the fields are disabled and we cannot check their content
    // It looks like we can use shadow DOM to catch it but too complicated for now
    cy.contains('Registration Endpoint')
      .click();
    if (checkLabels) {cy.checkMachRegLabel({
      machRegName: machRegName,
      labelName: 'myLabel1',
      labelValue: 'myLabelValue1'})};
    if (checkAnnotations) {cy.checkMachRegAnnotation({
      machRegName: machRegName,
      annotationName: 'myAnnotation1',
      annotationValue: 'myAnnotationValue1'});}
});

// Add Label to machine registration
Cypress.Commands.add('addMachRegLabel', ({labelName, labelValue}) => {
  cy.get('[data-testid="labels-and-annotations-block"]').contains('Registration Endpoint')
    .click();
  cy.get('[data-testid="add-label-mach-reg"] > .footer > .btn')
    .click();
  cy.get('[data-testid="add-label-mach-reg"] > .kv-container > .kv-item.key')
    .type(labelName);
  cy.get('[data-testid="add-label-mach-reg"] > .kv-container > .kv-item.value')
    .type(labelValue);
});

// Add Annotation to machine registration
Cypress.Commands.add('addMachRegAnnotation', ({annotationName, annotationValue}) => {
  cy.get('[data-testid="labels-and-annotations-block"]').contains('Registration Endpoint')
    .click();
  cy.get('[data-testid="add-annotation-mach-reg"] > .footer > .btn')
    .click();
  cy.get('[data-testid="add-annotation-mach-reg"] > .kv-container > .kv-item.key')
    .type(annotationName);
  cy.get('[data-testid="add-annotation-mach-reg"] > .kv-container > .kv-item.value')
    .type(annotationValue);
});

// Add Label to machine inventory
Cypress.Commands.add('addMachInvLabel', ({labelName, labelValue, useHardwareLabels=true}) => {
  cy.get('[data-testid="labels-and-annotations-block"]').contains('Inventory of Machines')
    .click();
  cy.get('[data-testid="add-label-mach-inv"] > .footer > .btn')
    .click();
  cy.get('[data-testid="add-label-mach-inv"] > .kv-container > .kv-item.key').type(labelName);
  cy.get('[data-testid="add-label-mach-inv"] > .kv-container > .kv-item.value').type(labelValue);
  if (useHardwareLabels) {
    var nthChildIndex = 7;
    for (const key in hwLabels) {
      const value = hwLabels[key];
      cy.get('[data-testid="add-label-mach-inv"] > .footer > .btn')
        .click();
      cy.get(`[data-testid="add-label-mach-inv"] > .kv-container > :nth-child(${nthChildIndex}) > input`).type(key);
      cy.get(`[data-testid="add-label-mach-inv"] > .kv-container > :nth-child(${nthChildIndex + 1}) > .no-resize`).type(value, {parseSpecialCharSequences: false});
      nthChildIndex += 3;
    };
  };
});

// Add Annotation to machine inventory
Cypress.Commands.add('addMachInvAnnotation', ({annotationName, annotationValue}) => {
  cy.get('[data-testid="labels-and-annotations-block"]').contains('Inventory of Machines')
    .click();
  cy.clickButton('Add Annotation');
  cy.get('[data-testid="add-annotation-mach-inv"] > .kv-container > .kv-item.key')
    .type(annotationName);
  cy.get('[data-testid="add-annotation-mach-inv"] > .kv-container > .kv-item.value')
    .type(annotationValue);
});

// Check machine inventory label in YAML
Cypress.Commands.add('checkMachInvLabel', ({machRegName, labelName, labelValue, userHardwareLabels=true, afterBoot=false}) => {
  if (afterBoot == false ) {
    cy.contains(machRegName)
      .click();
    cy.get('div.actions > .role-multi-action')
      .click()
    cy.contains('li', 'Edit YAML')
      .click();
    cy.contains('Registration Endpoint: '+ machRegName)
      .should('exist');
    cy.contains('[data-testid="yaml-editor-code-mirror"]', labelName + ': ' + labelValue);
    if (userHardwareLabels) {
      for (const key in hwLabels) {
        const value = hwLabels[key];
        cy.contains('[data-testid="yaml-editor-code-mirror"]', key +': ' + value);
      };
    };
    cy.clickButton('Cancel');
  } else {
      cy.contains('[data-testid="yaml-editor-code-mirror"]', labelName + ': ' + labelValue);
      if (userHardwareLabels) {
        for (const key in hwLabels) {
          const value = hwLabels[key];
          cy.contains('[data-testid="yaml-editor-code-mirror"]', key +': ');
      };
    };
  }
});

// Check machine registration label in YAML
Cypress.Commands.add('checkMachRegLabel', ({
  machRegName,
  labelName,
  labelValue}) => {
    cy.contains(machRegName)
      .click();
    cy.get('div.actions > .role-multi-action')
      .click()
    cy.contains('li', 'Edit YAML')
      .click();
    cy.contains('Registration Endpoint: '+ machRegName)
      .should('exist');
    cy.contains('[data-testid="yaml-editor-code-mirror"]', labelName + ': ' + labelValue);
    cy.clickButton('Cancel');
});

// Check machine registration annotation in YAML
Cypress.Commands.add('checkMachRegAnnotation', ({
  machRegName,
  annotationName,
  annotationValue}) => {
    cy.contains(machRegName)
      .click();
    cy.get('div.actions > .role-multi-action')
      .click()
    cy.contains('li', 'Edit YAML')
      .click();
    cy.contains('Registration Endpoint: '+ machRegName)
      .should('exist');
    cy.contains('[data-testid="yaml-editor-code-mirror"]', annotationName + ': ' + annotationValue);
    cy.clickButton('Cancel');
});

// Edit a machine registration
Cypress.Commands.add('editMachReg', ({
  machRegName,
  addLabel=false,
  addAnnotation=false,
  withYAML=false}) => {
    cy.contains(machRegName)
      .click();
    // Select the 3dots button and edit configuration
    cy.get('div.actions > .role-multi-action')
      .click()
    if (withYAML) {
      cy.contains('li', 'Edit YAML')
        .click();
      cy.contains('metadata')
        .click(0,0)
        .type('{end}{enter}  labels:{enter}  myLabel1: myLabelValue1');
      cy.contains('metadata')
        .click(0,0)
        .type('{end}{enter}  annotations:{enter}  myAnnotation1: myAnnotationValue1');
    } else {
      cy.contains('li', 'Edit Config')
        .click();
      if (addLabel) {cy.addMachRegLabel({labelName: 'myLabel1',
        labelValue: 'myLabelValue1',
        withYAML: withYAML})};
      if (addAnnotation) {cy.addMachRegAnnotation({annotationName: 'myAnnotation1',
        annotationValue: 'myAnnotationValue1',
        withYAML: withYAML})};
    }
});

// Delete a machine registration
Cypress.Commands.add('deleteMachReg', ({machRegName}) => {
  cy.contains('Registration Endpoint')
    .click();
  /*  This code cannot be used anymore for now because of
      https://github.com/rancher/elemental/issues/714
      As it is not a blocker, we need to bypass it.
      Instead of selecting resource to delete by name
      we select all resources.
  cy.contains(machRegName)
    .parent()
    .parent()
    .click();
  */
  cy.get('[width="30"] > .checkbox-outer-container')
    .click();
  cy.contains('[data-testid="sortable-table-promptRemove"]', 'Delete')
    .click()
  cy.confirmDelete();
  // Timeout should fix this issue https://github.com/rancher/elemental/issues/643
  cy.contains(machRegName, {timeout: 20000})
    .should('not.exist')
});

// Machine Inventory functions

// Import machine inventory
Cypress.Commands.add('importMachineInventory', ({
  machineInventoryFile,
  machineInventoryName}) => {
    cy.clickNavMenu(["Inventory of Machines"]);
    cy.get('[data-testid="masthead-create-yaml"]')
      .click();
    cy.clickButton('Read from File');
    cy.get('input[type="file"]')
      .attachFile({filePath: machineInventoryFile});
    cy.contains('[data-testid="action-button-async-button"]', 'Create')
      .click();
    cy.contains('Creating')
      .should('not.exist');
    cy.contains(machineInventoryName)
      .should('exist');
});

Cypress.Commands.add('checkFilter', ({
  filterName,
  testFilterOne,
  testFilterTwo,
  shouldNotMatch}) => {
    cy.clickNavMenu(["Inventory of Machines"]);
    cy.clickButton("Add Filter");
    cy.get('.advanced-search-box').type(filterName);
    cy.get('.bottom-block > .role-primary').click();
    (testFilterOne) ? cy.contains('test-filter-one').should('exist') : cy.contains('test-filter-one').should('not.exist');
    (testFilterTwo) ? cy.contains('test-filter-two').should('exist') : cy.contains('test-filter-two').should('not.exist');
    (shouldNotMatch) ? cy.contains('shouldnotmatch').should('exist') : cy.contains('shouldnotmatch').should('not.exist');
});
