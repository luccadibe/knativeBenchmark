package functions;

import io.quarkus.test.junit.QuarkusTest;
import org.junit.jupiter.api.Test;

import static io.restassured.RestAssured.given;
import static org.hamcrest.CoreMatchers.equalTo;

@QuarkusTest
public class FunctionTest {

    @Test
    public void testFunction() {
        // First call should return "true"
        given()
            .when().get("/")
            .then()
            .statusCode(200)
            .body(equalTo("true"));

        // Second call should return "false"
        given()
            .when().get("/")
            .then()
            .statusCode(200)
            .body(equalTo("false"));
    }
}