const qs = require("querystring");

const got = require("got");
const faker = require("faker");

const valueGenerators = [
	faker.random.word,
	faker.internet.url,
	faker.address.streetAddress,
	faker.commerce.productName,
	faker.name.findName,
	faker.phone.phoneNumber
];

const randomValue = () => {
	return faker.random.arrayElement(valueGenerators)();
};

const createBin = async () => {
	const response = await got("http://postb.in/api/bin", {
		method: "POST"
	});
	return JSON.parse(response.body).binId;
};

const generateUniqueWords = length => {
	const words = faker.lorem.words(length).split(" ");
	// Strip out duplicate words
	const uniqueWords = [...new Set(words)];
	if (uniqueWords.length !== length) {
		return generateUniqueWords(length);
	}
	return uniqueWords;
};

const generateBody = binId => {
	const method = faker.random.arrayElement(["GET", "POST"]);

	const propertyCount = faker.random.number({ min: 2, max: 4 });
	const keys = generateUniqueWords(propertyCount);
	const names = generateUniqueWords(propertyCount);

	const query = keys.reduce((accumulator, key, i) => {
		accumulator[key] = `{${names[i]}}`;
		return accumulator;
	}, {});
	if (faker.random.number({ min: 1, max: 3 }) === 1) {
		let staticKey = faker.lorem.word();
		while (keys.indexOf(staticKey) >= 0) {
			staticKey = faker.lorem.word();
		}
		query[staticKey] = qs.escape(randomValue());
	}
	const querystring = qs.unescape(qs.stringify(query));
	const url = `http://postb.in/${binId}?${querystring}`;

	const endpoint = { method, url };

	const dataCount = faker.random.number({ min: 1, max: 3 });
	const data = Array.from(new Array(dataCount), () => {
		const d = {};
		names.forEach(name => {
			if (faker.random.number({ min: 0, max: 5 }) !== 0) {
				d[name] = randomValue();
			}
		});
		return d;
	});

	return { endpoint, data };
};

const createRequest = (body, endpoint) => {
	return got(endpoint, {
		method: "POST",
		headers: {
			"Content-Type": "application/json"
		},
		body
	});
};

module.exports = {
	generateBody,
	createRequest,
	createBin
};
