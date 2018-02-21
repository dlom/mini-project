const qs = require("querystring");

const got = require("got");
const faker = require("faker");

const REQUESTS_TO_PERFORM = 1000;

const valueGenerators = [
	faker.random.word,
	faker.internet.url,
	faker.address.streetAddress,
	faker.commerce.productName,
	faker.name.findName,
	faker.phone.phoneNumber
];

const createBin = async () => {
	const response = await got("http://postb.in/api/bin", {
		method: "POST"
	});
	return JSON.parse(response.body).binId;
};

const generateBody = binId => {
	const method = faker.random.arrayElement(["GET", "POST"]);

	const propertyCount = faker.random.number({ min: 2, max: 4 });
	const keys = faker.lorem.words(propertyCount).split(" ");
	const names = faker.lorem.words(propertyCount).split(" ");

	const query = keys.reduce((accumulator, key, i) => {
		accumulator[key] = `{${names[i]}}`;
		return accumulator;
	}, {});
	const querystring = qs.unescape(qs.stringify(query));
	const url = `http://postb.in/${binId}?${querystring}`;

	const endpoint = { method, url };

	const dataCount = faker.random.number({ min: 1, max: 3 });
	const data = Array.from(new Array(dataCount), () => {
		const d = {};
		names.forEach(name => {
			if (faker.random.number({ min: 0, max: 5 }) !== 0) {
				d[name] = faker.random.arrayElement(valueGenerators)();
			}
		});
		return d;
	});

	return { endpoint, data };
};

const createRequest = body => {
	return got("http://localhost/ingest.php", {
		method: "POST",
		headers: {
			"Content-Type": "application/json"
		},
		body
	});
};

const performALotOfRequests = (binId, count) => {
	const promises = Array.from(new Array(count), () => {
		return JSON.stringify(generateBody(binId));
	}).map(body => {
		return createRequest(body);
	});
	return Promise.all(promises);
};

(async () => {
	try {
		const binId = await createBin();
		console.log(`http://postb.in/b/${binId}`);
		const results = await performALotOfRequests(binId, REQUESTS_TO_PERFORM);
		if (results.length <= 20) {
			results.forEach(result => {
				console.log(`${result.statusCode}: ${result.statusMessage}`);
			});
		}
	} catch (e) {
		console.error(e);
	}
})();
