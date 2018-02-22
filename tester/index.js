const { URL } = require("url");
const assert = require("assert");

const Docker = require("dockerode");
const DockerStreamCleanser = require("docker-stream-cleanser");
const JSONStream = require("JSONStream");
const got = require("got");
const pThrottle = require("p-throttle");

const { generateBody, createRequest, createBin } = require("./util.js");

const REQUESTS_TO_PERFORM = 1000;
const DELIVERY_SERVICE_NAME = "delivery";
const INGESTION_SERVICE_ENDPOINT = "http://localhost/ingest.php";

const getService = async name => {
	const docker = new Docker();
	const containersInfo = await docker.listContainers();
	const containerInfo = containersInfo.find(containerInfo => {
		return containerInfo.Labels["com.docker.compose.service"] === name;
	});
	return docker.getContainer(containerInfo.Id);
};

// Stream magic
const streamToArray = (length, stream) => {
	const processedStream = stream.pipe(new DockerStreamCleanser()).pipe(JSONStream.parse());
	const collector = [];

	return new Promise(resolve => {
		processedStream.on("data", data => {
			collector.push(data);
			if (collector.length >= length) {
				stream.destroy();
				resolve([...collector]);
			}
		});
	});
};

// Recreating the same algorithm that the delivery service uses
const convertPostbackToFinal = postback => {
	const parsed = new URL(postback.endpoint.url);

	const regex = /{(.*?)}/;
	const replaceKeys = {};
	const plainKeys = {};
	for (const [key, value] of parsed.searchParams) {
		const regexResult = value.match(regex);
		if (regexResult === null) {
			plainKeys[key] = value;
		} else {
			replaceKeys[key] = regexResult[1];
		}
	}

	return postback.data.map(data => {
		const result = {};
		for (const key of Object.keys(replaceKeys)) {
			const replace = replaceKeys[key];
			if (Object.prototype.hasOwnProperty.call(data, replace)) {
				result[key] = data[replace];
			} else {
				result[key] = "";
			}
		}
		for (const key of Object.keys(plainKeys)) {
			result[key] = plainKeys[key];
		}
		return result;
	}).map(body => {
		return {
			method: postback.endpoint.method,
			body
		};
	});
};

// Avoid DoSing the bin (1 request / 5 ms)
const throttledChecker = pThrottle((binId, reqId) => {
	return got(`http://postb.in/api/bin/${binId}/req/${reqId}`, {
		method: "GET",
		headers: {
			"Content-Type": "application/json"
		}
	});
}, 1, 5);

const check = async (binId, logs, theoretical) => {
	const results = await Promise.all(logs.map(log => {
		return throttledChecker(binId, log.responseBody);
	}));

	const compare = results.map(result => {
		const parsed = JSON.parse(result.body);
		const body = (Object.keys(parsed.query).length) ? parsed.query : parsed.body;
		return {
			method: parsed.method,
			body
		};
	});

	const sadMessage = "Postbacks were not successfully processed!";
	assert.deepEqual(new Set(compare), new Set(theoretical), sadMessage);
};

(async () => {
	try {
		// Create bin
		console.log("Creating bin...");
		const binId = await createBin();
		console.log(`http://postb.in/b/${binId}`);

		// Grab log stream
		console.log("Grabbing log output...");
		const container = await getService(DELIVERY_SERVICE_NAME);
		const stream = await container.attach({
			stream: true,
			stderr: true
		});

		// Calculate what the output from delivery *should* be
		console.log("Generating postbacks...");
		const bodies = Array.from(new Array(REQUESTS_TO_PERFORM), () => {
			return generateBody(binId);
		});
		const theoretical = bodies.map(convertPostbackToFinal).reduce((a, b) => {
			return a.concat(b);
		});

		// Send off postbacks to the ingestion service
		console.log("Sending postbacks to the ingestion service...");
		const requests = bodies.map(body => {
			return JSON.stringify(body);
		}).map(body => {
			return createRequest(body, INGESTION_SERVICE_ENDPOINT);
		});
		await Promise.all(requests);
		console.log("Done sending postbacks!");

		// Wait for the data to come back from the other end...
		console.log("Reading log stream...");
		const logs = await streamToArray(theoretical.length, stream);
		console.log("Done reading log stream!");

		// Assert that the logged results are equal to the theorized values
		console.log("Comparing results... (this might take a while)");
		await check(binId, logs, theoretical);
		console.log("Postbacks processed successfully!");

		// Perform calculations on results?
	} catch (e) {
		console.log(e);
	}
})();
