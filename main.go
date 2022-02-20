package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/gorilla/mux"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

func main() {
	port := flag.Int("p", 8082, "port to use")
	pg_url := flag.String("db", "", "pg db url")
	flag.Parse()
	r := mux.NewRouter()
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "nikki pong")
	})
	r.HandleFunc("/dashboard", DashboardHandler(*pg_url)).Methods("GET")
	r.HandleFunc("/signin", SignInHandler(*pg_url)).Methods("POST")
	r.Handle("/admin/constructs", ListConstructHandler(*pg_url)).Methods("GET")
	r.Handle("/admin/constructs", SetConstructHandler(*pg_url)).Methods("POST")
	r.Handle("/admin/constructs/{construct_id}", DeleteConstructHandler(*pg_url)).Methods("DELETE")
	r.Handle("/admin/facets", ListFacetHandler(*pg_url)).Methods("GET")
	r.Handle("/admin/facets", SetFacetHandler(*pg_url)).Methods("POST")
	r.Handle("/admin/facets/{facet_id}", DeleteFacetHandler(*pg_url)).Methods("DELETE")
	r.Handle("/admin/questions", ListQuestionHandler(*pg_url)).Methods("GET")
	r.Handle("/admin/questions", SetQuestionHandler(*pg_url)).Methods("POST")
	r.Handle("/admin/questions/{question_id}", DeleteQuestionHandler(*pg_url)).Methods("DELETE")
	r.Handle("/daily", AnswerDailyHandler(*pg_url)).Methods("POST")
	r.Handle("/daily", ListDailyQuestionsHandler(*pg_url)).Methods("GET")
	r.Handle("/manager/answers", ListMemberAnswersHandler(*pg_url)).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(".")))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), r))
}

func ListMemberAnswersHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			fmt.Println(err)
			return
		}

		answers, err := collateAnswers(ctx, conn, user.ID)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		json.NewEncoder(w).Encode(answers)
	}
}

func SignInHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		idToken := strings.ReplaceAll(strings.TrimPrefix(r.Header.Get("Authorization"), "Token"), " ", "")
		user, err := decodeIdToken(ctx, idToken)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		u, err := linkFirebase(ctx, conn, user)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		// default team and organization for prototype
		err = setPrototypeDefaults(ctx, conn, u.ID)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		json.NewEncoder(w).Encode(u)
	}
}

func setPrototypeDefaults(ctx context.Context, conn *pgx.Conn, userID int) error {
	query := `
		INSERT INTO members (user_id, team_id)
		VALUES ($1, 1)
		ON CONFLICT ON CONSTRAINT members_un
		DO NOTHING;
	`
	_, err := conn.Exec(ctx, query, userID)
	return err
}

func ListDailyQuestionsHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			fmt.Println(err)
			return
		}
		daily, err := dailyQuestions(ctx, conn, user.ID)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		json.NewEncoder(w).Encode(daily)
	}
}

func ListConstructHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		constructs, err := listConstructs(ctx, conn)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		response := map[string][]Construct{"data": constructs}
		json.NewEncoder(w).Encode(response)
	}
}

func ListFacetHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		list, err := listFacets(ctx, conn)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		response := map[string][]Facet{"data": list}
		json.NewEncoder(w).Encode(response)
	}
}

func ListQuestionHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		list, err := listQuestions(ctx, conn)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		response := map[string][]Question{"data": list}
		json.NewEncoder(w).Encode(response)
	}
}

func verifyUser(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *pgx.Conn) (User, error) {
	idToken := strings.ReplaceAll(strings.TrimPrefix(r.Header.Get("Authorization"), "Token"), " ", "")
	user, err := userFromToken(ctx, conn, idToken)
	if err != nil {
		fmt.Println(err)
		response := map[string]string{"error": err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return user, err
	}
	return user, nil
}

type ConstructRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func SetConstructHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var req ConstructRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			response := map[string]string{"error": "invalid request body"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}

		construct, err := setConstruct(ctx, conn, req.Slug, req.Name)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		json.NewEncoder(w).Encode(construct)
	}
}

func DeleteFacetHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		facet_id, err := strconv.Atoi(vars["facet_id"])
		if err != nil {
			response := map[string]string{"error": "facet_id must be of type int"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		err = deleteFacet(ctx, conn, facet_id)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

func DeleteConstructHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		construct_id, err := strconv.Atoi(vars["construct_id"])
		if err != nil {
			response := map[string]string{"error": "construct_id must be of type int"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		err = deleteConstruct(ctx, conn, construct_id)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

type DailyAnswer struct {
	QuestionID int `json:"question_id"`
	Answer     int `json:"answer"`
}

func AnswerDailyHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var answers []DailyAnswer
		err := json.NewDecoder(r.Body).Decode(&answers)
		if err != nil {
			response := map[string]string{"error": "answers are required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		err = answerQuestions(ctx, conn, answers, user.ID)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

func answerQuestions(ctx context.Context, conn *pgx.Conn, answers []DailyAnswer, userID int) error {
	query := `
		INSERT INTO answers (question_id, response, user_id)
		VALUES ($1, $2, $3)
	`
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(query, a.QuestionID, a.Answer, userID)
	}
	return commit(ctx, tx, batch)
}

func commit(ctx context.Context, tx pgx.Tx, batch *pgx.Batch) error {
	results := tx.SendBatch(ctx, batch)
	err := results.Close()
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			return fmt.Errorf("error in sending batch (%s): %s. Hint: %s. (detail: %s, type: %s) where: line %d position %d in routine %s - %w", pgerr.Code, pgerr.Message, pgerr.Hint, pgerr.Detail, pgerr.DataTypeName, pgerr.Line, pgerr.Position, pgerr.Routine, err)
		}
		return fmt.Errorf("error in sending batch: %w", err)
	}
	return tx.Commit(ctx)
}

func DeleteQuestionHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		question_id, err := strconv.Atoi(vars["question_id"])
		if err != nil {
			response := map[string]string{"error": "question_id must be of type int"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}
		err = deleteQuestion(ctx, conn, question_id)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

func SetFacetHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var req map[string]string
		err := json.NewDecoder(r.Body).Decode(&req)
		if _, ok := req["construct_id"]; !ok {
			response := map[string]string{"error": "construct_id is required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		construct_id, err := strconv.Atoi(req["construct_id"])
		if err != nil {
			response := map[string]string{"error": "facet_id must be of type int"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		if _, ok := req["name"]; !ok {
			response := map[string]string{"error": "name is required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		name := req["name"]

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}

		err = setFacet(ctx, conn, construct_id, name)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

func SetQuestionHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var req map[string]string
		err := json.NewDecoder(r.Body).Decode(&req)
		if _, ok := req["facet_id"]; !ok {
			response := map[string]string{"error": "facet_id is required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		facet_id, err := strconv.Atoi(req["facet_id"])
		if err != nil {
			response := map[string]string{"error": "facet_id must be of type int"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		if _, ok := req["statement"]; !ok {
			response := map[string]string{"error": "statement is required"}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
		statement := req["statement"]

		ctx := r.Context()
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)

		user, err := verifyUser(ctx, w, r, conn)
		if err != nil {
			return
		}
		if !user.IsAdmin {
			response := map[string]string{"error": "endpoint restricted to admins"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
			return
		}

		err = setQuestion(ctx, conn, facet_id, statement)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}
}

type Construct struct {
	ID        int       `json:"construct_id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Facet struct {
	ID        int       `json:"facet_id"`
	Construct string    `json:"construct"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Question struct {
	ID        int       `json:"question_id"`
	Facet     string    `json:"facet"`
	FacetID   int       `json:"facet_id"`
	Statement string    `json:"statement"`
	CreatedAt time.Time `json:"created_at"`
}

func dailyQuestions(ctx context.Context, conn *pgx.Conn, userID int) ([]Question, error) {
	query := `
		select d.question_id, d.statement
		from get_or_generate_dailies(1) d
		ORDER BY RANDOM();
	`
	var questions []Question
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return questions, err
	}
	for rows.Next() {
		var question Question
		err := rows.Scan(&question.ID, &question.Statement)
		if err != nil {
			return questions, err
		}
		questions = append(questions, question)
	}
	return questions, nil
}

func listConstructs(ctx context.Context, conn *pgx.Conn) ([]Construct, error) {
	query := `
		SELECT construct_id, slug, name, created_at FROM constructs;
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var constructs []Construct
	for rows.Next() {
		var construct Construct
		err := rows.Scan(&construct.ID, &construct.Slug, &construct.Name, &construct.CreatedAt)
		if err != nil {
			return nil, err
		}
		constructs = append(constructs, construct)
	}
	return constructs, nil
}

func listFacets(ctx context.Context, conn *pgx.Conn) ([]Facet, error) {
	query := `
		SELECT facet_id, c.slug, f.name, f.created_at 
		FROM facets f
		JOIN constructs c USING (construct_id);
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var facets []Facet
	for rows.Next() {
		var facet Facet
		err := rows.Scan(&facet.ID, &facet.Construct, &facet.Name, &facet.CreatedAt)
		if err != nil {
			return nil, err
		}
		facets = append(facets, facet)
	}
	return facets, nil
}

func listQuestions(ctx context.Context, conn *pgx.Conn) ([]Question, error) {
	query := `
		SELECT question_id, f.name, statement, q.created_at 
		FROM questions q
		JOIN facets f USING (facet_id);
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var questions []Question
	for rows.Next() {
		var question Question
		err := rows.Scan(&question.ID, &question.Facet, &question.Statement, &question.CreatedAt)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}
	return questions, nil
}

func setConstruct(ctx context.Context, conn *pgx.Conn, slug, name string) (Construct, error) {
	query := `
		INSERT INTO constructs (slug, name)
		VALUES($1, $2)
		ON CONFLICT ON CONSTRAINT constructs_slug_key
		DO UPDATE SET name = $2
		RETURNING construct_id, slug, name, created_at;
	`
	var construct Construct
	row := conn.QueryRow(ctx, query, slug, name)
	err := row.Scan(&construct.ID, &construct.Slug, &construct.Name, &construct.CreatedAt)
	return construct, err
}

func deleteConstruct(ctx context.Context, conn *pgx.Conn, construct_id int) error {
	query := `
		DELETE from constructs WHERE construct_id = $1;
	`
	_, err := conn.Exec(ctx, query, construct_id)
	return err
}

func deleteFacet(ctx context.Context, conn *pgx.Conn, facet_id int) error {
	query := `
		DELETE from facets WHERE facet_id = $1;
	`
	_, err := conn.Exec(ctx, query, facet_id)
	return err
}

func deleteQuestion(ctx context.Context, conn *pgx.Conn, question_id int) error {
	query := `
		DELETE from questions WHERE question_id = $1;
	`
	_, err := conn.Exec(ctx, query, question_id)
	return err
}

func setFacet(ctx context.Context, conn *pgx.Conn, construct_id int, name string) error {
	query := `
		INSERT INTO facets (construct_id, name)
		VALUES($1, $2);
	`
	_, err := conn.Exec(ctx, query, construct_id, name)
	return err
}

func setQuestion(ctx context.Context, conn *pgx.Conn, facet_id int, statement string) error {
	query := `
		INSERT INTO questions (facet_id, statement)
		VALUES($1, $2);
	`
	_, err := conn.Exec(ctx, query, facet_id, statement)
	return err
}

func DashboardHandler(pg_url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		idToken := strings.ReplaceAll(strings.TrimPrefix(r.Header.Get("Authorization"), "Token"), " ", "")
		conn, err := pgx.Connect(ctx, pg_url)
		if err != nil {
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		defer conn.Close(ctx)
		user, err := userFromToken(ctx, conn, idToken)
		if err != nil {
			fmt.Println(err)
			response := map[string]string{"error": err.Error()}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		if user.IsAdmin {

		}
	}
}

type User struct {
	ID          int
	Username    string
	FirebaseUID string
	IsAdmin     bool
	IsStaff     bool
}

func userFromToken(ctx context.Context, conn *pgx.Conn, token string) (User, error) {
	var u User
	user, err := decodeIdToken(ctx, token)
	if err != nil {
		return u, err
	}
	query := `
	SELECT user_id, firebase_uid, username, is_admin, is_staff
	FROM users 
	WHERE firebase_uid = $1
	`
	row := conn.QueryRow(ctx, query, user.UID)
	err = row.Scan(&u.ID, &u.FirebaseUID, &u.Username, &u.IsAdmin, &u.IsStaff)
	if err != nil {
		fmt.Println(err)
		return u, err
	}
	return u, nil
}

func linkFirebase(ctx context.Context, conn *pgx.Conn, user *auth.Token) (User, error) {
	email := ""
	for k, v := range user.Firebase.Identities {
		if k == "email" {
			email = fmt.Sprintf("%v", v)
			email = strings.TrimPrefix(email, "[")
			email = strings.TrimSuffix(email, "]")
			break
		}
	}
	query := `
		INSERT INTO users (username, firebase_uid)
		SELECT $1, $2
		ON CONFLICT ON CONSTRAINT users_firebase_uid_key
		DO UPDATE SET
			username=EXCLUDED.username 
		RETURNING user_id, username, firebase_uid, is_admin, is_staff;
	`
	row := conn.QueryRow(ctx, query, email, user.UID)
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.FirebaseUID, &u.IsAdmin, &u.IsStaff)
	if err != nil {
		return u, err
	}
	return u, nil
}

func decodeIdToken(ctx context.Context, token string) (*auth.Token, error) {
	opt := option.WithCredentialsFile("credentials.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, errors.Wrap(err, "error initializing app")
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting Auth client")
	}
	return client.VerifyIDToken(ctx, token)
}

type CollatedAnsweres struct {
	Team      string `json:"team"`
	Construct string `json:"construct"`
	Facet     string `json:"facet"`
	Total     int    `json:"total"`
	Count     int    `json:"count"`
	Date      string `json:"date"`
}

func collateAnswers(ctx context.Context, conn *pgx.Conn, userID int) ([]CollatedAnsweres, error) {
	query := `
		select t.name, c.name, f.name, 
		coalesce(sum(-a.response+8) filter (where not q.reverse_measurement), 0)+
		coalesce(sum(-a.response+8) filter (where q.reverse_measurement),0) as total,
		count (a.response),
		to_char(a.created_at, 'yyyy-mm-dd')
		from members m
		join answers a
			on a.user_id = m.user_id 
		join questions q using (question_id)
		join facets f using (facet_id)
		join constructs c using (construct_id)
		join teams t using (team_id)
		where team_id in (
			select m.team_id  
			from members m
			where is_manager
			and user_id = $1
		)
		group by t.name, c.construct_id, f.facet_id, to_char(a.created_at, 'yyyy-mm-dd');
	`
	rows, err := conn.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	var results []CollatedAnsweres
	for rows.Next() {
		var collated CollatedAnsweres
		err := rows.Scan(&collated.Team, &collated.Construct, &collated.Facet, &collated.Total, &collated.Count, &collated.Date)
		if err != nil {
			return nil, err
		}
		results = append(results, collated)
	}
	return results, nil

}
