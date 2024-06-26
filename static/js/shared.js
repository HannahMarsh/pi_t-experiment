async function loadHeader(headerPath) {
    try {
        const response = await fetch(headerPath);
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const headerHTML = await response.text();
        document.getElementById('header').innerHTML = headerHTML;
    } catch (error) {
        console.error('Error loading header:', error);
    }
}

async function fetchData(url) {
    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('Error fetching data:', error);
        throw error;
    }
}

async function fetchDataAndDisplay() {
    try {
        const data = await fetchData('/data');
        displayData(data);
        updateBooleanCells()
        updateClientNodeNames(data)
    } catch (error) {
        document.getElementById('data').textContent = ('Error loading data: ' + error.message);
        console.error('Error loading data:', error);
    }
}

function startFetchingData() {
    fetchDataAndDisplay();
    setInterval(fetchDataAndDisplay, 1000); // Update every second
}

function updateBooleanCells() {
    const booleanCells = document.querySelectorAll('td');
    booleanCells.forEach(cell => {
        if (cell.textContent.trim().toLowerCase() === 'true') {
            cell.classList.add('true');
            cell.classList.remove('false');
        } else if (cell.textContent.trim().toLowerCase() === 'false') {
            cell.classList.add('false');
            cell.classList.remove('true');
        }
    });
}

let names = {};

function updateClientNodeNames(data) {
    const cells = document.querySelectorAll('td');
    cells.forEach(cell => {
        if (cell.textContent.trim().toLowerCase().includes('http://')) {
            const displayInfo = getName(data, cell.textContent.trim().toLowerCase());
            if (displayInfo) {
                cell.textContent = displayInfo.name;
                cell.classList.add(displayInfo.class);
            }
        }
    });
}

function getName(data, url) {

    if (names[url]) {
        return names[url];
    }
    if (url === '') {
        return { ID: '', type: '', name: '', class: '', short: '' };
    }
    for (const [node, status] of Object.entries(data.Nodes)) {
        if (url.includes(node) || node.includes(url)) {
            const info = {
                ID: status.Node.ID,
                type: 'Node',
                name: `Node${status.Node.ID} (${status.Node.IsMixer ? 'mixer' : 'gatekeeper'})`,
                class: status.Node.IsMixer ? 'mixer' : 'gatekeeper',
                short: `Node ${status.Node.ID}`
            };
            names[url] = info;
            return info;
        }
    }
    for (const [client, status] of Object.entries(data.Clients)) {
        if (url.includes(client) || client.includes(url)) {
            const info = {
                ID: status.Client.ID,
                type: 'Client',
                name: `Client${status.Client.ID}`,
                class: 'client',
                short: `Client ${status.Client.ID}`
            };
            names[url] = info;
            return info;
        }
    }
    return null;
}

function formatRoutingPath(routingPath) {
    return routingPath.map((node, index) => {
        const label = (index === routingPath.length - 1) ? 'client' : 'node';
        const className = (index === routingPath.length - 1) ? 'client' : (node.IsMixer ? 'mixer' : 'gatekeeper');
        return `<span class="${className}">${label}${node.ID}</span>`;
    }).join(' → ');
}