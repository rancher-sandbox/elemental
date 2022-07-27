// Generic functions

// Log into Rancher
Cypress.Commands.add('login', (username = Cypress.env('username'), password = Cypress.env('password'), cacheSession = Cypress.env('cache_session')) => {
  const login = () => {
    let loginPath
    loginPath="/v3-public/localProviders/local*";
    cy.intercept('POST', loginPath).as('loginReq');
    
    cy.visit('/auth/login');

    cy.byLabel('Username')
      .focus()
      .type(username, {log: false});

    cy.byLabel('Password')
      .focus()
      .type(password, {log: false});

    cy.get('button').click();
    cy.wait('@loginReq');
    cy.contains("Getting Started", {timeout: 10000}).should('be.visible');
    } 

  if (cacheSession) {
    cy.session([username, password], login);
  } else {
    login();
  }
});

// Search fields by label
Cypress.Commands.add('byLabel', (label) => {
  cy.get('.labeled-input').contains(label).siblings('input');
});

// Search button by label
Cypress.Commands.add('clickButton', (label) => {
  cy.get('.btn').contains(label).click();
});

// Confirm the delete operation
Cypress.Commands.add('confirmDelete', () => {
  cy.get('.card-actions').contains('Delete').click();
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
    cy.get(label).focus().clear().type(value, {log: log});
  } else {
    cy.byLabel(label).focus().clear().type(value, {log: log});
  }
});

// Insert a key/value pair
Cypress.Commands.add('typeKeyValue', ({key, value}) => {
  cy.get(key).clear().type(value);
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

// Machine registration functions

// Create a machine registration
Cypress.Commands.add('createMachReg', ({machRegName, namespace='fleet-default', checkLabels=false, checkAnnotations=false}) => {
  cy.clickNavMenu(["Dashboard"]);
  cy.clickButton("Create Machine Registration");
  if (namespace != "fleet-default") {
    cy.get('div.vs__selected-options').eq(0).click()
    cy.get('li.vs__dropdown-option').contains('Create a New Namespace').click()
    cy.get(':nth-child(1) > .labeled-input').type(namespace);
    cy.focused().tab().tab().type(machRegName);
  } else {
    cy.typeValue({label: 'Name', value: machRegName});
  }

  if (checkLabels) {
    cy.clickButton('Add Label');
    cy.contains('.row', 'Labels').within(() => {
      cy.get('.kv-item.key').type('myLabel1');
      cy.get('.kv-item.value').type('myLabelValue1');
    })
  }

  if (checkAnnotations) {
    cy.clickButton('Add Annotation')
    cy.contains('.row', 'Annotations').within(() => {
      cy.get('.kv-item.key').type('myAnnot1');
      cy.get('.kv-item.value').type('myAnnotValue1');
    })
  }

  cy.clickButton("Create");

  // Make sure the machine registration is created and active
  cy.contains('.masthead', 'Machine Registration: '+ machRegName + ' Active').should('exist');

  // Check the namespace
  cy.contains('.masthead', 'Namespace: '+ namespace).should('exist');

  // Make sure there is an URL registration in the Registration URL block
  cy.contains('.mt-40 > .col', /https:\/\/.*elemental\/registration/);

  // Try to download the registration file and check it
  cy.clickButton("Download");
  cy.verifyDownload('generate_iso_image.zip');
  
  // Check labels
  // The field is disabled so we cannot check its content...
  // It looks like we can use shadow DOM to catch it but too complicated for now

  // Check annotations
  // Same reason as labels

  // Check Cloud configuration
  // Cannot be checked yet due to https://github.com/rancher/dashboard/issues/6458
});

// Delete a machine registration
Cypress.Commands.add('deleteMachReg', ({machRegName}) => {
  cy.contains('Machine Registrations').click();
  cy.contains(machRegName).parent().parent().click();
  cy.clickButton('Delete');
  cy.confirmDelete();
  cy.contains(machRegName).should('not.exist')
});

// Delete all machine registrations
Cypress.Commands.add('deleteAllMachReg', () => {  
  cy.clickButton('Manage Machine Registrations');
  cy.get('[width="30"] > .checkbox-outer-container').click();
  cy.clickButton('Delete');
  cy.confirmDelete();
  cy.contains('There are no rows to show');
});
