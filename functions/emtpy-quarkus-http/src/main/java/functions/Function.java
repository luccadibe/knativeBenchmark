package functions;

import io.quarkus.funqy.Funq;

public class Function {

    private boolean isCold = true;

    @Funq
    public String function(Input input) {
        String response = Boolean.toString(isCold);
        isCold = false;
        return response;  // Return String directly instead of Output object
    }
}